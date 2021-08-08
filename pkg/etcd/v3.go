package etcd

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/pkg/transport"
	"github.com/trinhdaiphuc/etcdkeeper/config"
	"google.golang.org/grpc"
	"log"
)

type ClientV3 struct {
	*clientv3.Client
	UserName string
	Password string
	Host     string
}

func newClientV3(user UserInfo) (*ClientV3, error) {
	var (
		endpoints = []string{user.Host}
		err       error
		cfg       = config.GetConfig()
	)

	// use tls if usetls is true
	var tlsConfig *tls.Config
	if cfg.UseTLS {
		tlsInfo := transport.TLSInfo{
			CertFile:      cfg.CertFile,
			KeyFile:       cfg.KeyFile,
			TrustedCAFile: cfg.CaFile,
		}
		tlsConfig, err = tlsInfo.ClientConfig()
		if err != nil {
			log.Println(err.Error())
		}
	}

	conf := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: cfg.ConnectTimeout,
		TLS:         tlsConfig,
		DialOptions: []grpc.DialOption{grpc.WithBlock()},
	}
	if cfg.UseAuth {
		conf.Username = user.Username
		conf.Password = user.Password
	}

	var c *clientv3.Client
	c, err = clientv3.New(conf)
	if err != nil {
		return nil, err
	}
	return &ClientV3{
		Client:   c,
		UserName: user.Username,
		Password: user.Password,
		Host:     user.Host,
	}, nil
}

func GetInfoV3(rootClient *ClientV3) (map[string]string, error) {
	info := make(map[string]string)
	status, err := rootClient.Status(context.Background(), rootClient.Host)
	if err != nil {
		return nil, err
	}
	mems, err := rootClient.MemberList(context.Background())
	if err != nil {
		return nil, err
	}
	kb := 1024
	mb := kb * 1024
	gb := mb * 1024
	var sizeStr string
	for _, m := range mems.Members {
		if m.ID == status.Leader {
			info["version"] = status.Version
			gn, rem1 := size(int(status.DbSize), gb)
			mn, rem2 := size(rem1, mb)
			kn, bn := size(rem2, kb)
			if sizeStr != "" {
				sizeStr += " "
			}
			if gn > 0 {
				info["size"] = fmt.Sprintf("%dG", gn)
			} else {
				if mn > 0 {
					info["size"] = fmt.Sprintf("%dM", mn)
				} else {
					if kn > 0 {
						info["size"] = fmt.Sprintf("%dK", kn)
					} else {
						info["size"] = fmt.Sprintf("%dByte", bn)
					}
				}
			}
			info["name"] = m.GetName()
			break
		}
	}
	return info, nil
}

func GetTTL(cli *ClientV3, lease int64) int64 {
	resp, err := cli.Lease.TimeToLive(context.Background(), clientv3.LeaseID(lease))
	if err != nil {
		return 0
	}
	if resp.TTL == -1 {
		return 0
	}
	return resp.TTL
}


func GetPermissionPrefix(user UserInfo, key string) ([][]string, error) {
	if !config.GetConfig().UseAuth {
		return [][]string{{key, "p"}}, nil // No auth return all
	} else {
		if user.Username == "root" {
			return [][]string{{key, "p"}}, nil
		}
		rootCli, err := GetClientV3(user)
		if err != nil {
			return nil, err
		}
		defer rootCli.Close()

		if resp, err := rootCli.UserList(context.Background()); err != nil {
			return nil, err
		} else {
			// Find user permissions
			set := make(map[string]string)
			for _, u := range resp.Users {
				if u == user.Username {
					ur, err := rootCli.UserGet(context.Background(), u)
					if err != nil {
						return nil, err
					}
					for _, r := range ur.Roles {
						rr, err := rootCli.RoleGet(context.Background(), r)
						if err != nil {
							return nil, err
						}
						for _, p := range rr.Perm {
							set[string(p.Key)] = string(p.RangeEnd)
						}
					}
					break
				}
			}
			var pers [][]string
			for k, v := range set {
				pers = append(pers, []string{k, v})
			}
			return pers, nil
		}
	}
}


func size(num int, unit int) (n, rem int) {
	return num / unit, num - (num/unit)*unit
}
