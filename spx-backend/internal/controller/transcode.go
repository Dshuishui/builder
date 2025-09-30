package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	// "path/filepath"
	"strings"
	"time"

	"github.com/goplus/builder/spx-backend/internal/log"
	qiniuStorage "github.com/qiniu/go-sdk/v7/storage"
)

// TranscodeRequest 转码请求
type TranscodeRequest struct {
    VideoURL string `json:"videoUrl"` // WebM 视频的 Kodo URL
}

// TranscodeResponse 转码响应
type TranscodeResponse struct {
    VideoURL string `json:"videoUrl"` // 转码后的 MP4 URL
}

// TranscodeVideo 转码视频（WebM -> MP4 with H.264）
func (ctrl *Controller) TranscodeVideo(ctx context.Context, req *TranscodeRequest) (*TranscodeResponse, error) {
    logger := log.GetReqLogger(ctx)
    logger.Printf("Starting transcode: %s", req.VideoURL)
    
    // 1. 下载 WebM 文件
    inputPath, err := ctrl.downloadFromKodo(ctx, req.VideoURL)
    if err != nil {
        return nil, fmt.Errorf("download failed: %w", err)
    }
    defer os.Remove(inputPath)
    
    // 2. 转码为 MP4
    outputPath, err := ctrl.transcodeToMP4(ctx, inputPath)
    if err != nil {
        return nil, fmt.Errorf("transcode failed: %w", err)
    }
    defer os.Remove(outputPath)
    
    // 3. 上传到 Kodo
    mp4URL, err := ctrl.uploadToKodo(ctx, outputPath)
    if err != nil {
        return nil, fmt.Errorf("upload failed: %w", err)
    }
    
    logger.Printf("Transcode completed: %s", mp4URL)
    return &TranscodeResponse{VideoURL: mp4URL}, nil
}

// 从 Kodo 下载文件
func (ctrl *Controller) downloadFromKodo(ctx context.Context, kodoURL string) (string, error) {
    // 将 kodo:// URL 转换为 HTTP URL
    httpURL := ctrl.kodoURLToHTTP(kodoURL)
    
    // 创建临时文件
    tmpFile, err := os.CreateTemp("", "input-*.webm")
    if err != nil {
        return "", err
    }
    defer tmpFile.Close()
    
    // 下载文件
    resp, err := http.Get(httpURL)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("download failed: status %d", resp.StatusCode)
    }
    
    _, err = io.Copy(tmpFile, resp.Body)
    return tmpFile.Name(), err
}

// 使用 FFmpeg 转码
func (ctrl *Controller) transcodeToMP4(ctx context.Context, inputPath string) (string, error) {
    outputPath := inputPath + ".mp4"
    
    cmd := exec.CommandContext(ctx, "ffmpeg",
        "-i", inputPath,
		"-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2",
        "-c:v", "libx264",      // H.264 编码
        "-preset", "fast",
        "-crf", "23",
        "-c:a", "aac",
        "-b:a", "128k",
        "-movflags", "+faststart",
        "-y",                   // 覆盖输出文件
        outputPath,
    )
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("ffmpeg error: %w, output: %s", err, string(output))
    }
    
    return outputPath, nil
}

// 上传到 Kodo（复用现有逻辑）
func (ctrl *Controller) uploadToKodo(ctx context.Context, filePath string) (string, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer file.Close()
    
    stat, err := file.Stat()
    if err != nil {
        return "", err
    }
    
    // 生成唯一的文件 key
    key := fmt.Sprintf("videos/%d.mp4", time.Now().Unix())
    
    // 创建上传 token（参考 GetUpInfo）
    putPolicy := qiniuStorage.PutPolicy{
        Scope: ctrl.kodo.bucket + ":" + key,
    }
    upToken := putPolicy.UploadToken(ctrl.kodo.cred)
    
    // 配置上传区域
    cfg := qiniuStorage.Config{}
    // 根据 ctrl.kodo.bucketRegion 设置区域
    // 例如：cfg.Zone = &qiniuStorage.ZoneHuadong
    
    formUploader := qiniuStorage.NewFormUploader(&cfg)
    ret := qiniuStorage.PutRet{}
    
    putExtra := qiniuStorage.PutExtra{}
    err = formUploader.Put(ctx, &ret, upToken, key, file, stat.Size(), &putExtra)
    if err != nil {
        return "", err
    }
    
    // 返回 kodo:// 格式的 URL
    return fmt.Sprintf("kodo://%s/%s", ctrl.kodo.bucket, ret.Key), nil
}
// 将 kodo:// URL 转换为 HTTP URL
func (ctrl *Controller) kodoURLToHTTP(kodoURL string) string {
    // kodo://bucket/key -> https://domain/key
    // 参考 MakeFileURLs 的实现
    u, _ := url.Parse(kodoURL)
    key := strings.TrimPrefix(u.Path, "/")
    return ctrl.kodo.baseUrl + "/" + key
}