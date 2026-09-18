import { message } from 'antd';
import { ApiRequestError } from '../types/api';

export function reportError(error: unknown, fallback = '操作未完成') {
  if (error instanceof ApiRequestError) {
    const trace = error.requestId ? `（请求 ${error.requestId.slice(0, 8)}）` : '';
    void message.error(`${error.message}${trace}`);
    return;
  }
  void message.error(fallback);
}
