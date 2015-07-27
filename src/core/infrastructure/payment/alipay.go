/**
 * Copyright 2015 @ S1N1 Team.
 * name : new
 * author : jarryliu
 * date : 2015-07-27 20:22
 * description :
 * history :
 */
package payment

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)


type AlipayParameters struct {
	InputCharset string  `json:"_input_charset"` //网站编码
	Body         string  `json:"body"`           //订单描述
	NotifyUrl    string  `json:"notify_url"`     //异步通知页面
	OutTradeNo   string  `json:"out_trade_no"`   //订单唯一id
	Partner      string  `json:"partner"`        //合作者身份ID
	PaymentType  uint8   `json:"payment_type"`   //支付类型 1：商品购买
	ReturnUrl    string  `json:"return_url"`     //回调url
	SellerEmail  string  `json:"seller_email"`   //卖家支付宝邮箱
	Service      string  `json:"service"`        //接口名称
	Subject      string  `json:"subject"`        //商品名称
	TotalFee     float32 `json:"total_fee"`      //总价
	Sign         string  `json:"sign"`           //签名，生成签名时忽略
	SignType     string  `json:"sign_type"`      //签名类型，生成签名时忽略
}

type AliPay struct {
	Partner string //合作者ID
	Key     string //合作者私钥
	Seller  string //网站卖家邮箱地址
}

/* 按照支付宝规则生成sign */
func (this *AliPay) Sign(param interface{}) string {
	//解析为字节数组
	paramBytes, err := json.Marshal(param)
	if err != nil {
		return ""
	}

	//重组字符串
	var sign string
	oldString := string(paramBytes)

	//为保证签名前特殊字符串没有被转码，这里解码一次
	oldString = strings.Replace(oldString, `\u003c`, "<", -1)
	oldString = strings.Replace(oldString, `\u003e`, ">", -1)

	//去除特殊标点
	oldString = strings.Replace(oldString, "\"", "", -1)
	oldString = strings.Replace(oldString, "{", "", -1)
	oldString = strings.Replace(oldString, "}", "", -1)
	paramArray := strings.Split(oldString, ",")

	for _, v := range paramArray {
		detail := strings.SplitN(v, ":", 2)
		//排除sign和sign_type
		if detail[0] != "sign" && detail[0] != "sign_type" {
			//total_fee转化为2位小数
			if detail[0] == "total_fee" {
				number, _ := strconv.ParseFloat(detail[1], 32)
				detail[1] = strconv.FormatFloat(number, 'f', 2, 64)
			}
			if sign == "" {
				sign = detail[0] + "=" + detail[1]
			} else {
				sign += "&" + detail[0] + "=" + detail[1]
			}
		}
	}

	//追加密钥
	sign += this.Key

	//md5加密
	m := md5.New()
	m.Write([]byte(sign))
	sign = hex.EncodeToString(m.Sum(nil))
	return sign
}

func (this *AliPay) CreateGateway(orderNo string, fee float32, subject,
	body, notifyUrl, returnUrl string) string {
	//实例化参数
	param := &AlipayParameters{}
	param.InputCharset = "utf-8"
	param.NotifyUrl = url.QueryEscape(notifyUrl)
	param.OutTradeNo = orderNo
	param.Partner = this.Partner
	param.PaymentType = 1
	param.ReturnUrl = url.QueryEscape(returnUrl)
	param.SellerEmail = this.Seller
	param.Service = "create_direct_pay_by_user"
	param.Subject = subject
	param.Body = body
	param.TotalFee = fee

	//生成签名
	sign := this.Sign(param)

	//追加参数
	param.Sign = sign
	param.SignType = "MD5"

	//生成自动提交form
	return `
		<form id="alipaysubmit" name="alipaysubmit" action="https://mapi.alipay.com/gateway.do?_input_charset=utf-8" method="get" style='display:none;'>
			<input type="hidden" name="_input_charset" value="` + param.InputCharset + `">
			<input type="hidden" name="body" value="` + param.Body + `">
			<input type="hidden" name="notify_url" value="` + param.NotifyUrl + `">
			<input type="hidden" name="out_trade_no" value="` + param.OutTradeNo + `">
			<input type="hidden" name="partner" value="` + param.Partner + `">
			<input type="hidden" name="payment_type" value="` + strconv.Itoa(int(param.PaymentType)) + `">
			<input type="hidden" name="return_url" value="` + param.ReturnUrl + `">
			<input type="hidden" name="seller_email" value="` + param.SellerEmail + `">
			<input type="hidden" name="service" value="` + param.Service + `">
			<input type="hidden" name="subject" value="` + param.Subject + `">
			<input type="hidden" name="total_fee" value="` + strconv.FormatFloat(float64(param.TotalFee), 'f', 2, 32) + `">
			<input type="hidden" name="sign" value="` + param.Sign + `">
			<input type="hidden" name="sign_type" value="` + param.SignType + `">
		</form>
		<script>
			document.forms['alipaysubmit'].submit();
		</script>
	`
}

/* 被动接收支付宝同步跳转的页面 */
func (this *AliPay) Return(r *http.Request)Result{
	var payResult Result

	//实例化参数
	param := map[string]string{
		"body":         "", //描述
		"buyer_email":  "", //买家账号
		"buyer_id":     "", //买家ID
		"exterface":    "",
		"is_success":   "", //交易是否成功
		"notify_id":    "", //通知校验id
		"notify_time":  "", //校验时间
		"notify_type":  "", //校验类型
		"out_trade_no": "", //在网站中唯一id
		"payment_type": "", //支付类型
		"seller_email": "", //卖家账号
		"seller_id":    "", //卖家id
		"subject":      "", //商品名称
		"total_fee":    "", //总价
		"trade_no":     "", //支付宝交易号
		"trade_status": "", //交易状态 TRADE_FINISHED或TRADE_SUCCESS表示交易成功
		"sign":         "", //签名
		"sign_type":    "", //签名类型
	}

	//解析表单内容，失败返回错误代码-3
	form := r.URL.Query()
	for k, _ := range param {
		param[k] = form.Get(k)
	}

	payResult.OrderNo = param["out_trade_no"]
	payResult.TradeNo = param["trade_no"]
	fee ,_ := strconv.ParseFloat(param["total_fee"],32)
	payResult.Fee = float32(fee)

	//如果最基本的网站交易号为空，返回错误代码-1
	if payResult.OrderNo == "" { //不存在交易号
		payResult.Status = -1
		return payResult
	}
	//生成签名
	sign := this.Sign(param)
	//对比签名是否相同
	if sign == param["sign"] { //只有相同才说明该订单成功了
		//判断订单是否已完成
		tradeStatus := param["trade_status"]
		if tradeStatus == "TRADE_FINISHED" || tradeStatus == "TRADE_SUCCESS" { //交易成功
			payResult.Status = StatusTradeSuccess
		} else { //交易未完成，返回错误代码-4
			payResult.Status = -4
		}
	} else { //签名认证失败，返回错误代码-2
		payResult.Status = -2
	}

	//位置错误类型-5
	if payResult.Status == 0 {
		payResult.Status = -5
	}

	return payResult
}
