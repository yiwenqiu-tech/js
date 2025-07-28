package logic

import (
	"encoding/json"
	"log"
	"os"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	v20230901 "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/hunyuan/v20230901"
)

// HunyuanStreamSDK 使用腾讯云官方Go SDK流式API
func HunyuanStreamSDK(messages []*v20230901.Message, model string, onDelta func(string)) error {
	credential := common.NewCredential(
		os.Getenv("TENCENTCLOUD_SECRETID"),
		os.Getenv("TENCENTCLOUD_SECRETKEY"),
	)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "hunyuan.tencentcloudapi.com"
	cpf.Debug = false
	client, err := v20230901.NewClient(credential, "", cpf)
	if err != nil {
		return err
	}

	req := v20230901.NewChatCompletionsRequest()
	req.Model = common.StringPtr(model)
	req.Messages = messages
	req.Stream = common.BoolPtr(true)

	resp, err := client.ChatCompletions(req)
	if err != nil {
		log.Printf("[HunyuanSDK] ChatCompletions error: %v\n", err)
	}
	for event := range resp.Events {
		var respParams v20230901.ChatCompletionsResponseParams
		err := json.Unmarshal(event.Data, &respParams)
		if err != nil {
			log.Printf("Unmarshal resp error: %v", err)
		}
		onDelta(*respParams.Choices[0].Delta.Content)
	}
	return err
}
