import { client } from './common'

/**
 * 转码请求参数
 */
export type TranscodeRequest = {
  /** WebM 视频的 Kodo URL (kodo://bucket/key 格式) */
  videoUrl: string
}

/**
 * 转码响应
 */
export type TranscodeResponse = {
  /** 转码后的 MP4 视频 URL (kodo://bucket/key 格式) */
  videoUrl: string
}

/**
 * 调用后端转码服务，将 WebM 格式转换为 MP4 格式
 * @param params 转码请求参数
 * @param signal 可选的 AbortSignal 用于取消请求
 * @returns 转码后的视频 URL
 */
export async function transcodeVideo(params: TranscodeRequest, signal?: AbortSignal) {
  return client.post('/transcode', params, { signal }) as Promise<TranscodeResponse>
}