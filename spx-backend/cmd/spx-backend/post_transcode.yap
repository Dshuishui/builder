// Transcode video from WebM to MP4.
//
// Request:
//   POST /transcode

import (
	"github.com/goplus/builder/spx-backend/internal/controller"
)

ctx := &Context

params := &controller.TranscodeRequest{}
if !parseJSON(ctx, params) {
	return
}

response, err := ctrl.TranscodeVideo(ctx.Context(), params)
if err != nil {
	replyWithInnerError(ctx, err)
	return
}

json response