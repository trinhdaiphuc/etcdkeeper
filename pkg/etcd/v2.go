package etcd

import (
	"context"
	"github.com/coreos/etcd/client"
	"github.com/trinhdaiphuc/etcdkeeper/config"
	"sort"
	"strings"
)

type ClientV2 struct {
	client.Client
	Username string
	Password string
	Host     string
}

func newClientV2(user UserInfo) (*ClientV2, error) {
	cfg := client.Config{
		Endpoints:               []string{user.Host},
		HeaderTimeoutPerRequest: config.GetConfig().ConnectTimeout,
	}
	if config.GetConfig().UseAuth {
		cfg.Username = user.Username
		cfg.Password = user.Password
	}

	c, err := client.New(cfg)
	if err != nil {
		return nil, err
	}
	return &ClientV2{
		Client:   c,
		Username: user.Username,
		Password: user.Password,
		Host:     user.Host,
	}, nil
}

func GetInfoV2(rootClient *ClientV2) (map[string]string, error) {
	info := make(map[string]string)
	ver, err := rootClient.GetVersion(context.Background())
	if err != nil {
		return nil, err
	}
	memberKapi := client.NewMembersAPI(rootClient)
	member, err := memberKapi.Leader(context.Background())
	if err != nil {
		return nil, err
	}
	info["version"] = ver.Server
	info["name"] = member.Name
	info["size"] = "unknow" // FIXME: How get?
	return info, nil
}

func GetNode(node *client.Node, selKey string, all map[int][]map[string]interface{}, min, max int) int {
	separator := config.GetConfig().Separator
	keys := strings.Split(node.Key, separator) // /foo/bar
	if len(keys) < min && strings.HasPrefix(node.Key, selKey) {
		return max
	}
	for i := range keys { // ["", "foo", "bar"]
		k := strings.Join(keys[0:i+1], separator)
		if k == "" {
			continue
		}
		nodeMap := map[string]interface{}{"key": k, "dir": true, "nodes": make([]map[string]interface{}, 0)}
		if k == node.Key {
			nodeMap["value"] = node.Value
			nodeMap["dir"] = node.Dir
			nodeMap["ttl"] = node.TTL
			nodeMap["createdIndex"] = node.CreatedIndex
			nodeMap["modifiedIndex"] = node.ModifiedIndex
		}
		keylevel := len(strings.Split(k, separator))
		if keylevel > max {
			max = keylevel
		}

		if _, ok := all[keylevel]; !ok {
			all[keylevel] = make([]map[string]interface{}, 0)
		}
		var isExist bool
		for _, n := range all[keylevel] {
			if n["key"].(string) == k {
				isExist = true
			}
		}
		if !isExist {
			all[keylevel] = append(all[keylevel], nodeMap)
		}
	}

	if len(node.Nodes) != 0 {
		for _, n := range node.Nodes {
			max = GetNode(n, selKey, all, min, max)
		}
	}
	return max
}

func GetPermissionPrefixV2(user UserInfo, key string) ([][]string, error) {
	if config.GetConfig().UseAuth {
		return [][]string{{key, "p"}}, nil // No auth return all
	} else {
		if user.Username == "root" {
			return [][]string{{key, "p"}}, nil
		}

		if !strings.HasPrefix(user.Host, "http://") {
			user.Host = "http://" + user.Host
		}
		rootCli, err := GetClientV2(user)
		if err != nil {
			return nil, err
		}
		rootUserKapi := client.NewAuthUserAPI(rootCli)
		rootRoleKapi := client.NewAuthRoleAPI(rootCli)

		if users, err := rootUserKapi.ListUsers(context.Background()); err != nil {
			return nil, err
		} else {
			// Find user permissions
			set := make(map[string]string)
			for _, u := range users {
				if u == user.Username {
					user, err := rootUserKapi.GetUser(context.Background(), u)
					if err != nil {
						return nil, err
					}
					for _, r := range user.Roles {
						role, err := rootRoleKapi.GetRole(context.Background(), r)
						if err != nil {
							return nil, err
						}
						for _, ks := range role.Permissions.KV.Read {
							var k string
							if strings.HasSuffix(ks, "*") {
								k = ks[:len(ks)-1]
								set[k] = "p"
							} else if strings.HasSuffix(ks, "/*") {
								k = ks[:len(ks)-2]
								set[k] = "c"
							} else {
								if _, ok := set[ks]; !ok {
									set[ks] = ""
								}
							}
						}
					}
					break
				}
			}
			var pers [][]string
			var ks []string
			for k := range set {
				ks = append(ks, k)
			}
			sort.Strings(ks)
			for _, k := range ks {
				pers = append(pers, []string{k, set[k]})
			}
			return pers, nil
		}
	}
}
