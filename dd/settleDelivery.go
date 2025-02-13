package dd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"net/http"
)

type SettleDelivery struct {
	DeliveryType            int      `json:"deliveryType"` // 1,极速达 2, 全城配 3, 物流配送
	DeliveryName            string   `json:"deliveryName"`
	DeliveryDesc            string   `json:"deliveryDesc"`
	ExpectArrivalTime       string   `json:"expectArrivalTime"`
	ExpectArrivalEndTime    string   `json:"expectArrivalEndTime"`
	StoreDeliveryTemplateId string   `json:"storeDeliveryTemplateId"`
	DeliveryModeIdList      []string `json:"deliveryModeIdList"`
	AreaBlockId             string   `json:"areaBlockId"`
	AreaBlockName           string   `json:"areaBlockName"`
	FirstPeriod             int      `json:"firstPeriod"`
}

func parseSettleDelivery(g gjson.Result) (error, SettleDelivery) {
	r := SettleDelivery{
		DeliveryType:            int(g.Get("deliveryType").Num),
		DeliveryName:            g.Get("deliveryName").Str,
		DeliveryDesc:            g.Get("deliveryDesc").Str,
		ExpectArrivalTime:       g.Get("expectArrivalTime").Str,
		ExpectArrivalEndTime:    g.Get("expectArrivalEndTime").Str,
		StoreDeliveryTemplateId: g.Get("storeDeliveryTemplateId").Str,
		AreaBlockId:             g.Get("AreaBlockId").Str,
		AreaBlockName:           g.Get("areaBlockName").Str,
		FirstPeriod:             int(g.Get("firstPeriod").Num),
	}

	for _, v := range g.Get("deliveryModeIdList").Array() {
		r.DeliveryModeIdList = append(r.DeliveryModeIdList, v.Str)
	}
	return nil, r
}

type SettleInfo struct {
	SaasId          string         `json:"saasId"`
	Uid             string         `json:"uid"`
	FloorId         int            `json:"floorId"`
	FloorName       string         `json:"floorName"`
	SettleDelivery  SettleDelivery `json:"settleDelivery"`
	DeliveryAddress Address        `json:"deliveryAddress"`
}

func (s *DingdongSession) GetSettleInfo(result gjson.Result) error {
	r := SettleInfo{}

	for _, v := range result.Get("data.settleDelivery").Array() {
		_, settleDelivery := parseSettleDelivery(v)
		r.SettleDelivery = settleDelivery

	}
	r.SaasId = result.Get("data.saasId").Str
	r.Uid = result.Get("data.uid").Str
	r.FloorId = int(result.Get("data.floorId").Num)
	r.FloorName = result.Get("data.floorName").Str
	address, err := parseAddress(result.Get("data.deliveryAddress"))
	if err == nil {
		r.DeliveryAddress = address
	}

	s.SettleInfo = r
	return nil
}

type StoreInfo struct {
	StoreId                 string `json:"storeId"`
	StoreType               string `json:"storeType"`
	AreaBlockId             string `json:"areaBlockId"`
	StoreDeliveryTemplateId string `json:"-"`
	DeliveryModeId          string `json:"-"`
}

type DeliveryInfoVO struct {
	StoreDeliveryTemplateId string `json:"storeDeliveryTemplateId"`
	DeliveryModeId          string `json:"deliveryModeId"`
	StoreType               string `json:"storeType"`
}

type SettleParam struct {
	Uid              string         `json:"uid"`
	AddressId        string         `json:"addressId"`
	DeliveryInfoVO   DeliveryInfoVO `json:"deliveryInfoVO"`
	CartDeliveryType int            `json:"cartDeliveryType"`
	StoreInfo        StoreInfo      `json:"storeInfo"`
	CouponList       []string       `json:"couponList"`
	IsSelfPickup     int            `json:"isSelfPickup"`
	FloorId          int            `json:"floorId"`
	GoodsList        []Goods        `json:"goodsList"`
}

func (s *DingdongSession) CheckSettleInfo() error {
	urlPath := "https://api-sams.walmartmobile.cn/api/v1/sams/trade/settlement/getSettleInfo"

	data := SettleParam{
		Uid:              s.Uid,
		AddressId:        s.Address.AddressId,
		DeliveryInfoVO:   s.DeliveryInfoVO,
		CartDeliveryType: 2,
		StoreInfo:        s.FloorInfo.StoreInfo,
		CouponList:       make([]string, 0),
		IsSelfPickup:     0,
		FloorId:          s.FloorId,
		GoodsList:        s.GoodsList,
	}

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
			return s.GetSettleInfo(result)
		case "LIMITED":
			return LimitedErr
		case "CART_GOOD_CHANGE":
			return CartGoodChangeErr
		default:
			return errors.New(result.Get("msg").Str)
		}
	} else {
		return errors.New(fmt.Sprintf("[%v] %s", resp.StatusCode, body))
	}
}
