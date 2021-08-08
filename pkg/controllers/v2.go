package controllers

import (
	"context"
	"encoding/json"
	"github.com/coreos/etcd/client"
	"github.com/labstack/echo/v4"
	"github.com/trinhdaiphuc/etcdkeeper/config"
	"github.com/trinhdaiphuc/etcdkeeper/pkg/etcd"
	"github.com/trinhdaiphuc/etcdkeeper/pkg/middlewares"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func ConnectV2(ctx echo.Context) error {
	host := strings.TrimSpace(ctx.FormValue("host"))
	username := ctx.FormValue("uname")
	password := ctx.FormValue("passwd")
	if !strings.HasPrefix(host, "http") {
		host = "http://" + host
	}

	if config.GetConfig().UseAuth {
		if username != "root" {
			return ctx.JSON(http.StatusOK, map[string]interface{}{"status": "root"})
		}
		if username == "" || password == "" {
			return ctx.JSON(http.StatusOK, map[string]interface{}{"status": "login"})
		}
	}

	userInfo := &etcd.UserInfo{
		Host:     host,
		Username: username,
		Password: password,
	}

	client, err := etcd.GetClientV2(*userInfo)
	if err != nil {
		return ctx.JSON(http.StatusOK, map[string]interface{}{"status": "error", "message": err.Error()})
	}

	info, err := etcd.GetInfoV2(client)
	if err != nil {
		ctx.JSON(http.StatusOK, map[string]interface{}{"status": "error", "message": err.Error()})
	}

	token, err := middlewares.NewToken(userInfo)
	if err != nil {
		ctx.JSON(http.StatusOK, map[string]interface{}{"status": "error", "message": err.Error()})
	}
	return ctx.JSON(http.StatusOK, map[string]interface{}{"status": "running", "info": info, "token": token})
}

func PutV2(ctx echo.Context) error {
	key := ctx.FormValue("key")
	value := ctx.FormValue("value")
	ttl := ctx.FormValue("ttl")
	dir := ctx.FormValue("dir")
	log.Println("PUT v2", key)

	userInfo, ok := middlewares.GetUserInfo(ctx)
	if !ok {
		return ctx.String(http.StatusOK, "Missing User's info. Login again")
	}
	cli, err := etcd.GetClientV2(*userInfo)
	if err != nil {
		ctx.String(http.StatusOK, err.Error())
	}
	kapi := client.NewKeysAPI(cli)

	var isDir bool
	if dir != "" {
		isDir, _ = strconv.ParseBool(dir)
	}

	data := make(map[string]interface{})
	if ttl != "" {
		var sec int64
		sec, err = strconv.ParseInt(ttl, 10, 64)
		if err != nil {
			log.Println(err.Error())
		}
		_, err = kapi.Set(context.Background(), key, value, &client.SetOptions{TTL: time.Duration(sec) * time.Second, Dir: isDir})
	} else {
		_, err = kapi.Set(context.Background(), key, value, &client.SetOptions{Dir: isDir})
	}
	if err != nil {
		data["errorCode"] = 500
		data["message"] = err.Error()
	} else {
		if resp, err := kapi.Get(context.Background(), key, &client.GetOptions{Recursive: true, Sort: true}); err != nil {
			data["errorCode"] = err.Error()
		} else {
			if resp.Node != nil {
				node := make(map[string]interface{})
				node["key"] = resp.Node.Key
				node["value"] = resp.Node.Value
				node["dir"] = resp.Node.Dir
				node["ttl"] = resp.Node.TTL
				node["createdIndex"] = resp.Node.CreatedIndex
				node["modifiedIndex"] = resp.Node.ModifiedIndex
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

func GetV2(ctx echo.Context) error {
	key := ctx.FormValue("key")
	data := make(map[string]interface{})
	separator := config.GetConfig().Separator
	log.Println("GET v2", key)

	userInfo, ok := middlewares.GetUserInfo(ctx)
	if !ok {
		return ctx.String(http.StatusOK, "Missing User's info. Login again")
	}

	cli, err := etcd.GetClientV2(*userInfo)
	if err != nil {
		ctx.JSON(http.StatusOK, err.Error())
	}
	kapi := client.NewKeysAPI(cli)

	var permissions [][]string
	if ctx.FormValue("prefix") == "true" {
		var e error
		permissions, e = etcd.GetPermissionPrefixV2(*userInfo, key)
		if e != nil {
			return ctx.String(http.StatusOK, e.Error())
		}
	} else {
		permissions = [][]string{{key, ""}}
	}

	var (
		min, max int
	)
	if key == separator {
		min = 1
	} else {
		min = len(strings.Split(key, separator))
	}
	max = min
	all := make(map[int][]map[string]interface{})
	if key == separator {
		all[min] = []map[string]interface{}{{"key": key, "value": "", "dir": true, "nodes": make([]map[string]interface{}, 0)}}
	}
	for _, p := range permissions {
		pKey, pRange := p[0], p[1]
		var opt *client.GetOptions
		if pRange != "" {
			if pRange == "c" {
				pKey += separator
			}
			opt = &client.GetOptions{Recursive: true, Sort: true}
		}
		if resp, err := kapi.Get(context.Background(), pKey, opt); err != nil {
			data["errorCode"] = 500
			data["message"] = err.Error()
		} else {
			if resp.Node == nil {
				data["errorCode"] = 500
				data["message"] = "The node does not exist."
			} else {
				max = etcd.GetNode(resp.Node, key, all, min, max)
			}
		}
	}

	// parent-child mapping
	for i := max; i > min; i-- {
		for _, a := range all[i] {
			for _, pa := range all[i-1] {
				if i == 2 { // The last is root
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

	for _, n := range all[min] {
		if n["key"] == key {
			etcd.NodesSort(n)
			data["node"] = n
			break
		}
	}

	var dataByte []byte
	if dataByte, err = json.Marshal(data); err != nil {
		ctx.String(http.StatusOK, err.Error())
	}
	return ctx.String(http.StatusOK, string(dataByte))
}

func DelV2(ctx echo.Context) error {
	key := ctx.FormValue("key")
	dir := ctx.FormValue("dir")
	log.Println("DELETE v2", key)

	userInfo, ok := middlewares.GetUserInfo(ctx)
	if !ok {
		return ctx.String(http.StatusOK, "Missing User's info. Login again")
	}
	cli, err := etcd.GetClientV2(*userInfo)
	if err != nil {
		ctx.String(http.StatusOK, err.Error())
	}
	kapi := client.NewKeysAPI(cli)

	isDir, _ := strconv.ParseBool(dir)
	if isDir {
		if _, err := kapi.Delete(context.Background(), key, &client.DeleteOptions{Recursive: true, Dir: true}); err != nil {
			return ctx.String(http.StatusOK, err.Error())
		}
	} else {
		if _, err := kapi.Delete(context.Background(), key, nil); err != nil {
			return ctx.String(http.StatusOK, err.Error())
		}
	}

	return ctx.String(http.StatusOK, "ok")
}

func GetPathV2(ctx echo.Context) error {
	return GetV2(ctx)
}
