package logic

import (
	"fmt"
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
	fmt.Printf("secretId: %v, secretKey: %v\n", os.Getenv("TENCENTCLOUD_SECRETID"), os.Getenv("TENCENTCLOUD_SECRETKEY"))
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "hunyuan.ap-guangzhou.tencentcloudapi.com"
	client, err := v20230901.NewClient(credential, "", cpf)
	if err != nil {
		fmt.Printf("NewClient: %v", err)
		return err
	}

	req := v20230901.NewChatCompletionsRequest()
	req.Model = common.StringPtr(model)
	req.Messages = messages
	req.Stream = common.BoolPtr(true)
	fmt.Printf("ChatCompletions Begin\n")
	resp, err := client.ChatCompletions(req)
	if err != nil {
		fmt.Printf("ChatCompletions: %v", err)
		return err
	}
	fmt.Printf("ChatCompletions End: %v\n", resp)
	fmt.Printf("ChatCompletions End: %v\n", resp != nil)
	fmt.Printf("ChatCompletions End: %v\n", resp.Response != nil)
	if resp != nil && resp.Response != nil && resp.Response.Choices != nil {
		for _, choice := range resp.Response.Choices {
			if choice.Delta != nil && choice.Delta.Content != nil {
				onDelta(*choice.Delta.Content)
			}
		}
	}
	return nil
}
