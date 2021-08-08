![image](https://github.com/evildecay/etcdkeeper/blob/master/logo/logo-horizontal.png)

## ETCD Keeper

* Lightweight etcd web client.
* Support etcd 2.x and etcd 3.x.
* The server uses the etcd go client interface, and the server compiles with the etcd client package.
* Based easyui framework to achieve(easyui license [easyui website](http://www.jeasyui.com)).

## Usage

* Run etcdkeeper by go

```shell
go run main.go
```

* Run etcdkeeper by Docker-compose (with etcd-server)

```shell
docker-compose up
```  

* Add environment USE_AUTH=true (If enable etcd authentication)

You can custom config with environments:

```dotenv
HOST=127.0.0.1                  // host name or ip address (default: "0.0.0.0", the http server addreess, not etcd address)
PORT=8080                       // port (default 8080)
SEPARATOR=/                     // Separator (default "/")
USE_TLS=false                   // use tls (only v3, defaul false)
CA_FILE=path/to/ca-file         //verify certificates of TLS-enabled secure servers using this CA bundle (only v3)
CERT_FILE=path/to/cert-file     // identify secure client using this TLS certificate file (only v3)
KEY_FILE=path/to/key-file       // identify secure client using this TLS key file (only v3)
USE_AUTH=true                   // use etcd auth (default false)
CONNECT_TIMEOUT=5s              // ETCD client connect timeout (default 5s) such as "300ms", "-1.5h" or "2h45m".
                                // Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
SECRET_KEY=secret               // Jwt secret key (defaul secret)                  
EXPIRED_TIME=24h                // Jwt expired time (defaul 24h)
```

* Open your browser and enter the address: http://127.0.0.1:8080
* Click on the version of the title to select the version of ETCD. The default is V3. Reopening will remember your
  choice.
* Right click on the tree node to add or delete.
* Get data based on etcd user permissions.
    - Just display the list according to the configured permissions, and there will be time to add the configuration
      permission features.
    - Each time you restart etcdkeeper, you need to enter the root username and password for each etcd server address.
    - [enable etcdv3 authentication](https://etcd.io/docs/v3.2/op-guide/authentication/)
    - [enable etcdv2 authentication](https://etcd.io/docs/v2.3/authentication/)
* Display the status information of etcd, version, data size.
* Etcd address can be modified by default to the localhost. If you change, press the Enter key to take effect.

## Features

* Etcd client view, Add, update or delete nodes.
* Content edits use the ace editor([Ace editor](https://ace.c9.io)). Support toml,ini,yaml,json,xml and so on to
  highlight view.
* Content format. (Currently only support json, Other types can be extended later) Thanks jim3ma for his
  contribution.[@jim3ma]( https://github.com/jim3ma)

## Work in progress

* Add import and export features.  **(delay)**

## Special Note

Because the etcdv3 version uses the new storage concept, without the catalog concept, the client uses the previous
default "/" delimiter to view. See the documentation for
etcdv3 [clientv3 doc](https://godoc.org/github.com/coreos/etcd/clientv3).

## Docker

My Etdkeeper image. (https://hub.docker.com/r/bigphuc/etcdkeeper)

## Screenshots

![image](./screenshots/ui.png)

## Demo

![image](./screenshots/ui.gif)

## License

MIT
