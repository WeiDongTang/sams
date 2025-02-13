package dd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"net/http"
	"time"
)

type CapCityResponse struct {
	StrDate        string `json:"strDate"`
	DeliveryDesc   string `json:"deliveryDesc"`
	DeliveryDescEn string `json:"deliveryDescEn"`
	DateISFull     bool   `json:"dateISFull"`
}

type Capacity struct {
	Data                      string            `json:"data"`
	CapCityResponseList       []CapCityResponse `json:"capcityResponseList"`
	PortalPerformanceTemplate string            `json:"getPortalPerformanceTemplateResponse"`
}

func parseCapacity(g gjson.Result) (error, CapCityResponse) {
	var sizes []map[string]interface{}
	for _, size := range g.Get("sizes").Array() {
		sizes = append(sizes, size.Value().(map[string]interface{}))
	}
	capacity := CapCityResponse{
		StrDate:        g.Get("strDate").Str,
		DeliveryDesc:   g.Get("deliveryDesc").Str,
		DeliveryDescEn: g.Get("deliveryDescEn").Str,
		DateISFull:     g.Get("dateISFull").Bool(),
	}
	return nil, capacity
}



func (s *DingdongSession) GetCapacity(result gjson.Result) error {
	var capCityResponseList []CapCityResponse
	for _, v := range result.Get("data.capcityResponseList").Array() {
		_, product := parseCapacity(v)
		capCityResponseList = append(capCityResponseList, product)
	}
	s.Capacity = Capacity{
		Data:                      result.String(),
		CapCityResponseList:       capCityResponseList,
		PortalPerformanceTemplate: result.Get("data.getPortalPerformanceTemplateResponse").Str,
	}
	return nil
}

func (s *DingdongSession) CheckCapacity() error {
	urlPath := "https://api-sams.walmartmobile.cn/api/v1/sams/delivery/portal/getCapacityData"

	data := make(map[string]interface{})
	data["perDateList"] = []string{time.Now().Format("2006-01-02"), time.Now().AddDate(0, 0, 1).Format("2006-01-02")}
	data["storeDeliveryTemplateId"] = "703398195375534614"
	dataStr, _ := json.Marshal(data)

	req, _ := http.NewRequest("POST", urlPath, bytes.NewReader(dataStr))
	req.Header.Set("Host", "api-sams.walmartmobile.cn")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "*/*")
	//req.Header.Set("auth-token", "xxxxxxxxxxxx")
	req.Header.Set("auth-token", s.AuthToken)
	//req.Header.Set("app-version", "5.0.46.1")
	req.Header.Set("device-type", "ios")
	req.Header.Set("Accept-Language", "zh-Hans-CN;q=1, en-CN;q=0.9, ga-IE;q=0.8")
	//req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	//req.Header.Set("apptype", "ios")
	//req.Header.Set("device-name", "iPhone12,8")
	//req.Header.Set("device-os-version", "13.4.1")
	req.Header.Set("User-Agent", "SamClub/5.0.46 (iPhone; iOS 13.4.1; Scale/2.00)")
	req.Header.Set("system-language", "CN")

	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	resp.Body.Close()
	if resp.StatusCode == 200 {
		result := gjson.Parse(string(body))
		switch result.Get("code").Str {
		case "Success":
			return s.GetCapacity(result)
		default:
			return errors.New(result.Get("msg").Str)
		}
	} else {
		return errors.New(fmt.Sprintf("[%v] %s", resp.StatusCode, body))
	}
}
