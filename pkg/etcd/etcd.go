package etcd

import (
	"fmt"
	"sync"
)

type UserInfo struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
}

var (
	userMapV2 sync.Map
	userMapV3 sync.Map
)

func GetClientV2(user UserInfo) (*ClientV2, error) {
	instance, ok := userMapV2.Load(user)
	if ok {
		clientV2, _ := instance.(*ClientV2)
		if user.Username != clientV2.Username || user.Password != clientV2.Password {
			client, err := newClientV2(user)
			if err != nil {
				return nil, err
			}
			userMapV2.Store(user.Host, client)
			return client, nil
		}
		return instance.(*ClientV2), nil
	}
	client, err := newClientV2(user)
	if err != nil {
		return nil, err
	}
	instance, _ = userMapV2.LoadOrStore(user, client)
	return instance.(*ClientV2), nil
}

func GetClientV3(user UserInfo) (*ClientV3, error) {
	instance, ok := userMapV3.Load(user)
	if ok {
		clientV3, _ := instance.(*ClientV3)
		if user.Username != clientV3.UserName || user.Password != clientV3.Password {
			clientV3.Close()
			client, err := newClientV3(user)
			fmt.Println("Update client")
			if err != nil {
				return nil, err
			}
			userMapV2.Store(user, client)
			return client, nil
		}
		return instance.(*ClientV3), nil
	}
	client, err := newClientV3(user)
	if err != nil {
		return nil, err
	}
	fmt.Println("New client")
	instance, _ = userMapV3.LoadOrStore(user, client)
	return instance.(*ClientV3), nil
}

func NodesSort(node map[string]interface{}) {
	if v, ok := node["nodes"]; ok && v != nil {
		a := v.([]map[string]interface{})
		if len(a) != 0 {
			for i := 0; i < len(a)-1; i++ {
				NodesSort(a[i])
				for j := i + 1; j < len(a); j++ {
					if a[j]["key"].(string) < a[i]["key"].(string) {
						a[i], a[j] = a[j], a[i]
					}
				}
			}
			NodesSort(a[len(a)-1])
		}
	}
}
