import { Modal, Input, Typography } from 'antd';

interface Props {
  open: boolean;
  title: string;
  consequence: string;
  confirmLabel: string;
  noteLabel?: string;
  note: string;
  requireNote?: boolean;
  busy?: boolean;
  onNoteChange(value: string): void;
  onCancel(): void;
  onConfirm(): void;
}

export function ConfirmActionDialog({ open, title, consequence, confirmLabel, noteLabel = '复核说明', note, requireNote = false, busy = false, onNoteChange, onCancel, onConfirm }: Props) {
  return (
    <Modal
      open={open}
      title={title}
      okText={confirmLabel}
      cancelText="返回检查"
      okButtonProps={{ disabled: requireNote && note.trim().length < 4, loading: busy }}
      onOk={onConfirm}
      onCancel={onCancel}
      destroyOnClose
    >
      <Typography.Paragraph className="dialog-consequence">{consequence}</Typography.Paragraph>
      <label className="field-label" htmlFor="confirmation-note">{noteLabel}{requireNote ? '（必填）' : ''}</label>
      <Input.TextArea id="confirmation-note" value={note} rows={4} maxLength={500} showCount onChange={(event) => onNoteChange(event.target.value)} />
    </Modal>
  );
}
