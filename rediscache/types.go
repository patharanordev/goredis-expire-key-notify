package rediscache

type (
	ReqCheckout struct {
		Id     string `json:"id"`
		Expire int64  `json:"expire"`
	}
	OrderInfo struct {
		TTL int `json:"ttl"`
	}
	Info struct {
		TTL   int64  `json:"ttl"`
		Value string `json:"value"`
	}
	ResponseObj struct {
		Status int     `json:"status"`
		Data   *Info   `json:"data"`
		Error  *string `json:"error"`
	}
)
