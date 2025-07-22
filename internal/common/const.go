package common

import "os"

const (
	RolePrompt = "你是一位成瘾治疗心理医生，主要治疗用户性成瘾的问题，包括自慰、看黄等问题；请根据用户的输入给用户建议与帮助，帮助用户逐步戒掉性成瘾的问题。"
)

var HunyuanToken string
var HunyuanModel = "hunyuan-turbos-latest"
var HunyuanBaseUrl = "https://api.hunyuan.cloud.tencent.com/v1"

func init() {
	HunyuanToken = os.Getenv("HUNYUAN_TOKEN")
	if len(HunyuanToken) == 0 {
		panic("ENV OF HUNYUAN_TOKEN IS EMPTY")
	}
}
