package controllers

import (
	"context"
	"encoding/json"
	"github.com/coreos/etcd/clientv3"
	"github.com/labstack/echo/v4"
	"github.com/trinhdaiphuc/etcdkeeper/config"
	"github.com/trinhdaiphuc/etcdkeeper/pkg/etcd"
	"github.com/trinhdaiphuc/etcdkeeper/pkg/middlewares"
	"net/http"
	"strconv"
	"strings"
)

type Request struct {
	Key string `param:"key" query:"key" form:"key" json:"key"`
	Dir bool   `param:"dir" query:"dir" form:"dir" json:"dir"`
}

func GetSeparator(ctx echo.Context) error {
	return ctx.String(200, config.GetConfig().Separator)
}

func ConnectV3(ctx echo.Context) error {
	var err error
	if err != nil {
		return ctx.String(http.StatusOK, err.Error())
	}
	host := strings.TrimSpace(ctx.FormValue("host"))
	username := ctx.FormValue("uname")
	password := ctx.FormValue("passwd")
	if !strings.HasPrefix(host, "http") {
		host = "http://" + host
	}

	if config.GetConfig().UseAuth {
		if username == "" || password == "" {
			return ctx.JSON(http.StatusOK, map[string]interface{}{"status": "login"})
		}
		if username != "root" {
			return ctx.JSON(http.StatusOK, map[string]interface{}{"status": "root"})
		}
	}

	userInfo := &etcd.UserInfo{
		Host:     host,
		Username: username,
		Password: password,
	}

	client, err := etcd.GetClientV3(*userInfo)
	if err != nil {
		return ctx.JSON(http.StatusOK, map[string]interface{}{"status": "error", "message": err.Error()})
	}

	info, err := etcd.GetInfoV3(client)
	if err != nil {
		ctx.JSON(http.StatusOK, map[string]interface{}{"status": "error", "message": err.Error()})
	}

	token, err := middlewares.NewToken(userInfo)
	if err != nil {
		ctx.JSON(http.StatusOK, map[string]interface{}{"status": "error", "message": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{"status": "running", "info": info, "token": token})
}

func PutV3(ctx echo.Context) error {
	key := ctx.FormValue("key")
	value := ctx.FormValue("value")
	ttl := ctx.FormValue("ttl")
	userInfo, ok := middlewares.GetUserInfo(ctx)
	if !ok {
		return ctx.String(http.StatusOK, "Missing User's info. Login again")
	}
	cli, err := etcd.GetClientV3(*userInfo)
	if err != nil {
		ctx.String(http.StatusOK, err.Error())
	}

	data := make(map[string]interface{})
	if ttl != "" {
		var sec int64
		sec, err = strconv.ParseInt(ttl, 10, 64)
		if err != nil {
			ctx.Logger().Error(err)
		}
		var leaseResp *clientv3.LeaseGrantResponse
		leaseResp, err = cli.Grant(context.TODO(), sec)
		if err == nil && leaseResp != nil {
			_, err = cli.Put(context.Background(), key, value, clientv3.WithLease(leaseResp.ID))
		}
	} else {
		_, err = cli.Put(context.Background(), key, value)
	}
	if err != nil {
		data["errorCode"] = 500
		data["message"] = err.Error()
	} else {
		if resp, err := cli.Get(context.Background(), key); err != nil {
			data["errorCode"] = 500
			data["errorCode"] = err.Error()
		} else {
			if resp.Count > 0 {
				kv := resp.Kvs[0]
				node := make(map[string]interface{})
				node["key"] = string(kv.Key)
				node["value"] = string(kv.Value)
				node["dir"] = false
				node["ttl"] = etcd.GetTTL(cli, kv.Lease)
				node["createdIndex"] = kv.CreateRevision
				node["modifiedIndex"] = kv.ModRevision
				data["node"] = node
			}
		}
	}

	var dataByte []byte
	if dataByte, err = json.Marshal(data); err != nil {
		return ctx.String(http.StatusOK, err.Error())
	}
	return ctx.String(http.StatusOK, string(dataByte))
}

func GetV3(ctx echo.Context) error {
	data := make(map[string]interface{})
	key := ctx.FormValue("key")
	ctx.Logger().Info("GET v3", key)

	userInfo, ok := middlewares.GetUserInfo(ctx)
	if !ok {
		return ctx.String(http.StatusOK, "Missing User's info. Login again")
	}

	cli, err := etcd.GetClientV3(*userInfo)
	if err != nil {
		ctx.JSON(http.StatusOK, err.Error())
	}

	permissions, err := etcd.GetPermissionPrefix(*userInfo, key)
	if err != nil {
		return ctx.String(http.StatusOK, err.Error())
	}

	if ctx.FormValue("prefix") == "true" {
		pnode := make(map[string]interface{})
		pnode["key"] = key
		pnode["nodes"] = make([]map[string]interface{}, 0)
		for _, p := range permissions {
			var (
				resp *clientv3.GetResponse
				err  error
			)
			if p[1] != "" {
				prefixKey := p[0]
				if p[0] == "/" {
					prefixKey = ""
				}
				resp, err = cli.Get(context.Background(), prefixKey, clientv3.WithPrefix())
			} else {
				resp, err = cli.Get(context.Background(), p[0])
			}
			if err != nil {
				data["errorCode"] = 500
				data["message"] = err.Error()
			} else {
				for _, kv := range resp.Kvs {
					node := make(map[string]interface{})
					node["key"] = string(kv.Key)
					node["value"] = string(kv.Value)
					node["dir"] = false
					if key == string(kv.Key) {
						node["ttl"] = etcd.GetTTL(cli, kv.Lease)
					} else {
						node["ttl"] = 0
					}
					node["createdIndex"] = kv.CreateRevision
					node["modifiedIndex"] = kv.ModRevision
					nodes := pnode["nodes"].([]map[string]interface{})
					pnode["nodes"] = append(nodes, node)
				}
			}
		}
		data["node"] = pnode
	} else {
		if resp, err := cli.Get(context.Background(), key); err != nil {
			data["errorCode"] = 500
			data["message"] = err.Error()
		} else {
			if resp.Count > 0 {
				kv := resp.Kvs[0]
				node := make(map[string]interface{})
				node["key"] = string(kv.Key)
				node["value"] = string(kv.Value)
				node["dir"] = false
				node["ttl"] = etcd.GetTTL(cli, kv.Lease)
				node["createdIndex"] = kv.CreateRevision
				node["modifiedIndex"] = kv.ModRevision
				data["node"] = node
			} else {
				data["errorCode"] = 500
				data["message"] = "The node does not exist."
			}
		}
	}

	var dataByte []byte
	if dataByte, err = json.Marshal(data); err != nil {
		return ctx.String(http.StatusOK, err.Error())
	}
	return ctx.String(http.StatusOK, string(dataByte))
}

func GetPathV3(ctx echo.Context) error {
	var (
		data      = make(map[string]interface{})
		dataByte  []byte
		separator = config.GetConfig().Separator
		/*
			{1:["/"], 2:["/foo", "/foo2"], 3:["/foo/bar", "/foo2/bar"], 4:["/foo/bar/test"]}
		*/
		all       = make(map[int][]map[string]interface{})
		min       int
		max       int
		originKey = ctx.FormValue("key")
		// parent
		presp *clientv3.GetResponse
	)
	ctx.Logger().Info("GET v3", originKey)

	userInfo, ok := middlewares.GetUserInfo(ctx)
	if !ok {
		return ctx.String(http.StatusOK, "Missing User's info. Login again")
	}

	cli, err := etcd.GetClientV3(*userInfo)
	if err != nil {
		return ctx.String(http.StatusOK, err.Error())
	}

	permissions, err := etcd.GetPermissionPrefix(*userInfo, originKey)
	if err != nil {
		return ctx.String(http.StatusOK, err.Error())
	}

	if originKey != separator {
		presp, err = cli.Get(context.Background(), originKey)
		if err != nil {
			data["errorCode"] = 500
			data["message"] = err.Error()
			dataByte, _ = json.Marshal(data)
			return ctx.String(http.StatusOK, string(dataByte))

		}
	}
	if originKey == separator {
		min = 1
		//prefixKey = separator
	} else {
		min = len(strings.Split(originKey, separator))
		//prefixKey = originKey
	}
	max = min
	all[min] = []map[string]interface{}{{"key": originKey}}
	if presp != nil && presp.Count != 0 {
		all[min][0]["value"] = string(presp.Kvs[0].Value)
		all[min][0]["ttl"] = etcd.GetTTL(cli, presp.Kvs[0].Lease)
		all[min][0]["createdIndex"] = presp.Kvs[0].CreateRevision
		all[min][0]["modifiedIndex"] = presp.Kvs[0].ModRevision
	}
	all[min][0]["nodes"] = make([]map[string]interface{}, 0)

	for _, p := range permissions {
		key, rangeEnd := p[0], p[1]
		//child
		var resp *clientv3.GetResponse
		if rangeEnd != "" {
			resp, err = cli.Get(context.Background(), key, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
		} else {
			resp, err = cli.Get(context.Background(), key, clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
		}
		if err != nil {
			data["errorCode"] = 500
			data["message"] = err.Error()
			dataByte, _ = json.Marshal(data)
			return ctx.String(http.StatusOK, string(dataByte))
		}

		for _, kv := range resp.Kvs {
			if string(kv.Key) == separator {
				continue
			}
			keys := strings.Split(string(kv.Key), separator) // /foo/bar
			for i := range keys {                            // ["", "foo", "bar"]
				k := strings.Join(keys[0:i+1], separator)
				if k == "" {
					continue
				}
				node := map[string]interface{}{"key": k}
				if node["key"].(string) == string(kv.Key) {
					node["value"] = string(kv.Value)
					if key == string(kv.Key) {
						node["ttl"] = etcd.GetTTL(cli, kv.Lease)
					} else {
						node["ttl"] = 0
					}
					node["createdIndex"] = kv.CreateRevision
					node["modifiedIndex"] = kv.ModRevision
				}
				level := len(strings.Split(k, separator))
				if level > max {
					max = level
				}

				if _, ok := all[level]; !ok {
					all[level] = make([]map[string]interface{}, 0)
				}
				levelNodes := all[level]
				var isExist bool
				for _, n := range levelNodes {
					if n["key"].(string) == k {
						isExist = true
					}
				}
				if !isExist {
					node["nodes"] = make([]map[string]interface{}, 0)
					all[level] = append(all[level], node)
				}
			}
		}
	}

	// parent-child mapping
	for i := max; i > min; i-- {
		for _, a := range all[i] {
			for _, pa := range all[i-1] {
				if i == 2 {
					pa["nodes"] = append(pa["nodes"].([]map[string]interface{}), a)
					pa["dir"] = true
				} else {
					if strings.HasPrefix(a["key"].(string), pa["key"].(string)+separator) {
						pa["nodes"] = append(pa["nodes"].([]map[string]interface{}), a)
						pa["dir"] = true
					}
				}
			}
		}
	}
	data = all[min][0]

	if dataByte, err = json.Marshal(map[string]interface{}{"node": data}); err != nil {
		return ctx.String(http.StatusOK, err.Error())
	}
	return ctx.String(http.StatusOK, string(dataByte))
}

func DelV3(ctx echo.Context) error {
	var (
		request   Request
		separator = config.GetConfig().Separator
	)
	err := ctx.Bind(&request)
	if err != nil {
		return ctx.String(http.StatusOK, err.Error())
	}

	userInfo, ok := middlewares.GetUserInfo(ctx)
	if !ok {
		return ctx.String(http.StatusOK, "Missing User's info. Login again")
	}

	cli, err := etcd.GetClientV3(*userInfo)
	if err != nil {
		return ctx.String(http.StatusOK, err.Error())
	}

	ctx.Logger().Info("DELETE v3", request.Key)

	if _, err = cli.Delete(context.Background(), request.Key); err != nil {
		return ctx.String(http.StatusOK, err.Error())
	}

	if request.Dir {
		if _, err = cli.Delete(context.Background(), request.Key+separator, clientv3.WithPrefix()); err != nil {
			return ctx.String(http.StatusOK, err.Error())
		}
	}
	return ctx.String(http.StatusOK, "ok")
}
