package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/model"
)

type Config struct {
	Port            string
	DBDriver        string
	DBDSN           string
	JWTSecret       string
	JWTTTL          time.Duration
	CORSOrigins     []string
	LogLevel        slog.Level
	AutoMigrate     bool
	SeedData        bool
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Port:            env("PORT", "8080"),
		DBDriver:        strings.ToLower(env("DB_DRIVER", "postgres")),
		JWTSecret:       env("JWT_SECRET", "local-development-secret-change-me-521"),
		JWTTTL:          durationEnv("JWT_TTL", 8*time.Hour),
		CORSOrigins:     splitCSV(env("CORS_ORIGINS", "http://localhost:18521,http://127.0.0.1:18521")),
		LogLevel:        parseLogLevel(env("LOG_LEVEL", "info")),
		AutoMigrate:     boolEnv("DB_AUTO_MIGRATE", true),
		SeedData:        boolEnv("SEED_DATA", true),
		ShutdownTimeout: durationEnv("SHUTDOWN_TIMEOUT", 8*time.Second),
	}

	switch cfg.DBDriver {
	case "sqlite":
		cfg.DBDSN = env("DB_DSN", "file:mine-ventilation?mode=memory&cache=shared")
	case "postgres":
		cfg.DBDSN = env("DB_DSN", postgresDSN())
	default:
		return Config{}, fmt.Errorf("unsupported DB_DRIVER %q", cfg.DBDriver)
	}

	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	if _, err := strconv.Atoi(cfg.Port); err != nil {
		return Config{}, fmt.Errorf("invalid PORT %q: %w", cfg.Port, err)
	}
	if len(cfg.CORSOrigins) == 0 {
		return Config{}, fmt.Errorf("CORS_ORIGINS must not be empty")
	}
	return cfg, nil
}

func OpenDatabase(cfg Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	if cfg.DBDriver == "sqlite" {
		dialector = sqlite.Open(cfg.DBDSN)
	} else {
		dialector = postgres.Open(cfg.DBDSN)
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Warn),
		DisableForeignKeyConstraintWhenMigrating: false,
		TranslateError:                           true,
	})
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", cfg.DBDriver, err)
	}
	if cfg.AutoMigrate {
		if err := db.AutoMigrate(
			&model.User{},
			&model.VentilationNode{},
			&model.AirwayEdge{},
			&model.FanScenario{},
			&model.SimulationRun{},
			&model.AuditEvent{},
		); err != nil {
			return nil, fmt.Errorf("auto migrate: %w", err)
		}
	}
	if cfg.SeedData {
		if err := seed(db); err != nil {
			return nil, fmt.Errorf("seed database: %w", err)
		}
	}
	return db, nil
}

func ConfigureLogger(level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func seed(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		users := []model.User{
			{Email: "engineer@mine.local", DisplayName: "通风工程师", Role: string(constants.RoleEngineer), Active: true},
			{Email: "reviewer@mine.local", DisplayName: "安全复核员", Role: string(constants.RoleReviewer), Active: true},
			{Email: "admin@mine.local", DisplayName: "系统管理员", Role: string(constants.RoleAdmin), Active: true},
		}
		for i := range users {
			hash, err := bcrypt.GenerateFromPassword([]byte("Ventilate!2026"), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("hash seed password: %w", err)
			}
			users[i].PasswordHash = string(hash)
			var existing model.User
			if err := tx.Where("email = ?", users[i].Email).First(&existing).Error; err == nil {
				users[i] = existing
				continue
			} else if err != nil && err != gorm.ErrRecordNotFound {
				return err
			}
			if err := tx.Create(&users[i]).Error; err != nil {
				return err
			}
		}

		var count int64
		if err := tx.Model(&model.VentilationNode{}).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		nodes := []model.VentilationNode{
			{Code: "INT-01", NodeType: string(constants.NodeTypeIntake), ElevationM: 12, PressurePa: 1250, Status: string(constants.NodeStatusActive)},
			{Code: "JCT-12", NodeType: string(constants.NodeTypeJunction), ElevationM: -85, PressurePa: 680, Status: string(constants.NodeStatusActive)},
			{Code: "WF-07", NodeType: string(constants.NodeTypeWorkface), ElevationM: -126, RequiredAirflowM3S: 18, PressurePa: 410, Status: string(constants.NodeStatusActive)},
			{Code: "EXT-02", NodeType: string(constants.NodeTypeExhaust), ElevationM: 6, PressurePa: 0, Status: string(constants.NodeStatusActive)},
		}
		if err := tx.Create(&nodes).Error; err != nil {
			return err
		}
		edges := []model.AirwayEdge{
			{Code: "AW-101", FromNodeID: nodes[0].ID, ToNodeID: nodes[1].ID, ResistanceNS2M8: 1.8, AreaM2: 8.2, MaxVelocityMS: 8, DoorState: string(constants.DoorStateOpen), Enabled: true, CriticalPath: true, Version: 1},
			{Code: "AW-102", FromNodeID: nodes[1].ID, ToNodeID: nodes[2].ID, ResistanceNS2M8: 2.4, AreaM2: 6.4, MaxVelocityMS: 7, DoorState: string(constants.DoorStateOpen), Enabled: true, CriticalPath: true, Version: 1},
			{Code: "AW-103", FromNodeID: nodes[2].ID, ToNodeID: nodes[3].ID, ResistanceNS2M8: 2.1, AreaM2: 7.0, MaxVelocityMS: 8, DoorState: string(constants.DoorStateOpen), Enabled: true, CriticalPath: true, Version: 1},
			{Code: "AW-104", FromNodeID: nodes[1].ID, ToNodeID: nodes[3].ID, ResistanceNS2M8: 3.6, AreaM2: 5.8, MaxVelocityMS: 6, DoorState: string(constants.DoorStateRegulate), Enabled: true, Version: 1},
		}
		if err := tx.Create(&edges).Error; err != nil {
			return err
		}
		curve := datatypes.JSON([]byte(`[{"flow_m3s":0,"pressure_pa":1450},{"flow_m3s":30,"pressure_pa":1180},{"flow_m3s":60,"pressure_pa":720}]`))
		scenarios := []model.FanScenario{
			{Name: "夜班基准方案", Description: "当前网络的基准风机曲线，用于离线比较。", FanCurveJSON: curve, OperatingMode: "normal", ScenarioStatus: string(constants.ScenarioStatusApproved), SolverTolerance: 0.02, MaxIterations: 100, Version: 2, CreatedBy: users[0].ID, ApprovedBy: &users[1].ID},
			{Name: "检修降载草案", Description: "检修窗口的降载边界，仅供工程师提交复核。", FanCurveJSON: curve, OperatingMode: "reduced", ScenarioStatus: string(constants.ScenarioStatusDraft), SolverTolerance: 0.03, MaxIterations: 120, Version: 1, CreatedBy: users[0].ID},
		}
		if err := tx.Create(&scenarios).Error; err != nil {
			return err
		}
		return tx.Create(&model.AuditEvent{
			RequestID: "seed-bootstrap", ActorID: users[2].ID, ActorEmail: users[2].Email,
			Action: "system.seeded", EntityType: "system", EntityID: 1,
			BeforeState: `{}`, AfterState: `{"nodes":4,"edges":4,"scenarios":2}`,
			Metadata: `{"source":"bootstrap"}`, CreatedAt: time.Now().UTC(),
		}).Error
	})
}

func postgresDSN() string {
	host := env("DB_HOST", "127.0.0.1")
	port := env("DB_INTERNAL_PORT", "5432")
	name := env("DB_NAME", "ventilation")
	user := env("DB_USER", "ventilation")
	password := env("DB_PASSWORD", "ventilation_dev_password")
	sslMode := env("DB_SSLMODE", "disable")
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC", host, port, user, password, name, sslMode)
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func boolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseLogLevel(value string) slog.Level {
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
