package main

import (
	"bufio"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Knetic/govaluate"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func InitAddressMap(routerFile string) []Router {
	// Open our jsonFile
	jsonFile, err := os.Open(routerFile)
	// if we os.Open returns an error then handle it
	if err != nil {
		panic(err)
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var result = make([]Router, 0, 10)
	err = json.Unmarshal(byteValue, &result)
	if err != nil {
		panic(err)
	}

	return result
}

type Properties map[string]string

func InitDefaultConf() (Properties, error) {
	config := Properties{}
	filename := "conf.properties"

	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if equal := strings.Index(line, "="); equal >= 0 {
			if key := strings.TrimSpace(line[:equal]); len(key) > 0 {
				value := ""
				if len(line) > equal {
					value = strings.TrimSpace(line[equal+1:])
				}
				if value == "" {
					panic("conf.properties: " + key + "must not be empty!")
				}
				config[key] = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
		return nil, err
	}

	return config, nil
}

func main() {
	properties, err := InitDefaultConf()
	if err != nil {
		panic(nil)
	}
	var routerFile = flag.String("r", properties["default_routerFile"], "配置路由文件，默认router.json")
	var port = flag.String("p", properties["default_port"], "配置启动端口，默认8080")
	var callbackKey = flag.String("k", properties["default_callbackKey"], "配置回调解密key")
	var defaultUrl = properties["default_forward_url"]
	flag.Parse()
	log.Printf("load router file: %s\n", *routerFile)

	address := InitAddressMap(*routerFile)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover() //内置函数，可以捕捉到函数异常
			if err != nil {
				log.Println("err错误信息：", err)
				WriteError(w, "err msg: "+fmt.Errorf("%v", err).Error())
			}
		}()

		/* 验签 */
		{
			unescape, err := url.QueryUnescape(r.FormValue("data"))
			if err != nil {
				w.WriteHeader(500)
				WriteError(w, "urlDecode error")
				return
			}
			unescape = strings.Replace(unescape, " ", "+", -1)
			hashString := []byte("data=" + unescape + "&" + *callbackKey)
			hash := md5.Sum(hashString)
			sign := hex.EncodeToString(hash[:])
			if sign != r.FormValue("sign") {
				w.WriteHeader(500)
				WriteError(w, "sign error")
				return
			}
		}

		data, _ := base64.StdEncoding.DecodeString(r.FormValue("data"))
		var params Callback
		err := json.Unmarshal(data, &params)
		if err != nil {
			log.Println(err)
			WriteError(w, "json Unmarshal error")
			return
		}

		parameters := CallbackToMap(params)
		u := defaultUrl
		/* 根据表达式判判断是否需要转发到对应的URL上 */
		for _, router := range address {
			expression, _ := govaluate.NewEvaluableExpression(router.Expression)
			result, _ := expression.Evaluate(parameters)
			if result.(bool) {
				u = router.Url
				break
			}
		}

		log.Printf("forward to %s, data = %s\n", u, data)
		resp, err := http.PostForm(u, r.Form)
		if err != nil {
			log.Println(err)
			WriteError(w, "forward error")
			return
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Println("close io.ReadCloser error")
			}
		}(resp.Body)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			WriteError(w, "forward error, get resp body error")
			return
		}
		w.WriteHeader(resp.StatusCode)
		_, err = w.Write(body)
		if err != nil {
			log.Println(err)
			return
		}
	})

	log.Printf("starting server on http://localhost:%s/\n", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

func WriteError(w http.ResponseWriter, body string) {
	w.WriteHeader(500)
	_, err := w.Write([]byte(body))
	if err != nil {
		log.Println(err)
		return
	}
}

func test() {
	/* amount=1.0 "amount==1" "amount==1.0" "amount==1.00" 结果都为true */
	//expression, _ := govaluate.NewEvaluableExpression("amount==1.00")
	//expression, _ := govaluate.NewEvaluableExpression("(amount==1) && selfDefine=='透传参数'")
	//expression, _ := govaluate.NewEvaluableExpression("(amount==2) && selfDefine=='透传参数'")
	//expression, _ := govaluate.NewEvaluableExpression("(amount==2) || selfDefine=='透传参数'")
	//expression, _ := govaluate.NewEvaluableExpression("(amount==2) || selfDefine=='透传参数 '")
	/* 强类型比较 字符串1!=数字1 */
	//expression, _ := govaluate.NewEvaluableExpression("goodsId==1")
	//expression, _ := govaluate.NewEvaluableExpression("goodsId=='1'")
	//result, _ := expression.Evaluate(parameters)
	//println(result.(bool))
}

type Router struct {
	Url        string `json:"url"`
	Expression string `json:"expression"`
}

type Callback struct {
	Amount                      float64 `json:"amount"`
	GameOrder                   string  `json:"gameOrder"`
	OrderNo                     string  `json:"orderNo"`
	Status                      int32   `json:"status"`
	SelfDefine                  string  `json:"selfDefine"`
	ChannelUid                  string  `json:"channelUid"`
	PayTime                     string  `json:"payTime"`
	Channel                     string  `json:"channel"`
	ChannelId                   int32   `json:"channelId"`
	GoodsId                     string  `json:"goodsId"`
	GoodsName                   string  `json:"goodsName"`
	Yx_is_in_intro_offer_period string  `json:"yx_is_in_intro_offer_period"`
	Yx_is_trial_period          string  `json:"yx_is_trial_period"`
	Iap_sub_expire              string  `json:"iap_sub_expire"`
	Iap_sub                     string  `json:"iap_sub"`
	Paytype                     string  `json:"paytype"`
	Yx_sub_type                 string  `json:"yx_sub_type"`
	DealAmount                  string  `json:"dealAmount"`
	QkChannelId                 int32   `json:"qkChannelId"`
	QuickChannelId              int32   `json:"quickChannelId"`
	Sandbox                     string  `json:"sandbox"`
	IapSub                      string  `json:"iapSub"`
	IapSubExpire                string  `json:"iapSubExpire"`
	Currency                    string  `json:"currency"`
	PayType                     string  `json:"payType"`
}

func CallbackToMap(callback Callback) map[string]interface{} {
	parameters := make(map[string]interface{})
	parameters["amount"] = callback.Amount
	parameters["gameOrder"] = callback.GameOrder
	parameters["orderNo"] = callback.OrderNo
	parameters["status"] = callback.Status
	parameters["selfDefine"] = callback.SelfDefine
	parameters["channelUid"] = callback.ChannelUid
	parameters["payTime"] = callback.PayTime
	parameters["channel"] = callback.Channel
	parameters["channelId"] = callback.ChannelId
	parameters["goodsId"] = callback.GoodsId
	parameters["goodsName"] = callback.GoodsName
	parameters["yx_is_in_intro_offer_period"] = callback.Yx_is_in_intro_offer_period
	parameters["yx_is_trial_period"] = callback.Yx_is_trial_period
	parameters["iap_sub_expire"] = callback.Iap_sub_expire
	parameters["iap_sub"] = callback.Iap_sub
	parameters["paytype"] = callback.Paytype
	parameters["yx_sub_type"] = callback.Yx_sub_type
	parameters["dealAmount"] = callback.DealAmount
	parameters["qkChannelId"] = callback.QkChannelId
	parameters["quickChannelId"] = callback.QuickChannelId
	parameters["sandbox"] = callback.Sandbox
	parameters["iapSub"] = callback.IapSub
	parameters["iapSubExpire"] = callback.IapSubExpire
	parameters["currency"] = callback.Currency
	parameters["payType"] = callback.PayType
	return parameters
}
