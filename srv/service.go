package srv

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/ability-sh/abi-lib/dynamic"
	"github.com/ability-sh/abi-lib/iid"
	"github.com/ability-sh/abi-micro/micro"
	"github.com/ability-sh/abi-micro/mongodb"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	DB_VER = 1
)

type AppService struct {
	config     interface{} `json:"-"`
	name       string      `json:"-"`
	Prefix     string      `json:"prefix"`
	BasePath   string      `json:"basePath"`
	Aid        int64       `json:"aid"`          //区域ID
	Nid        int64       `json:"nid"`          //节点ID
	Expires    int64       `json:"expires"`      //过期秒数
	Db         string      `json:"db"`           // mongodb db
	AppMaxSize int64       `json:"app-max-size"` // 应用包最大字节数
	IID        *iid.IID    `json:"-"`
}

func newAppService(name string, config interface{}) *AppService {
	return &AppService{name: name, config: config}
}

/**
* 服务名称
**/
func (s *AppService) Name() string {
	return s.name
}

/**
* 服务配置
**/
func (s *AppService) Config() interface{} {
	return s.config
}

/**
* 初始化服务
**/
func (s *AppService) OnInit(ctx micro.Context) error {

	dynamic.SetValue(s, s.config)

	s.IID = iid.NewIID(s.Aid, s.Nid)

	ctx.Printf("db init ...")

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return err
	}

	db := conn.Database(s.Db)

	c := context.Background()

	db_app := db.Collection("app")
	db_ver := db.Collection("ver")
	db_container := db.Collection("container")
	db_ac := db.Collection("ac")

	{
		indexes := db_app.Indexes()
		_, err = indexes.CreateMany(c, []mongo.IndexModel{
			{
				Keys: bson.D{bson.E{"ctime", -1}},
			},
		})
		if err != nil {
			return err
		}
	}

	{
		indexes := db_ver.Indexes()
		_, err = indexes.CreateMany(c, []mongo.IndexModel{
			{
				Keys:    bson.D{bson.E{"appid", -1}, bson.E{"ver", -1}},
				Options: options.Index().SetUnique(true),
			},
			{
				Keys: bson.D{bson.E{"ctime", -1}},
			},
		})
		if err != nil {
			return err
		}
	}

	{
		indexes := db_container.Indexes()
		_, err = indexes.CreateMany(c, []mongo.IndexModel{
			{
				Keys: bson.D{bson.E{"ctime", -1}},
			},
		})
		if err != nil {
			return err
		}
	}

	{
		indexes := db_ac.Indexes()
		_, err = indexes.CreateMany(c, []mongo.IndexModel{
			{
				Keys:    bson.D{bson.E{"cid", -1}, bson.E{"appid", -1}},
				Options: options.Index().SetUnique(true),
			},
			{
				Keys: bson.D{bson.E{"ctime", -1}},
			},
		})
		if err != nil {
			return err
		}
	}

	ctx.Printf("db init done")

	return nil
}

/**
* 校验服务是否可用
**/
func (s *AppService) OnValid(ctx micro.Context) error {
	return nil
}

func (s *AppService) Recycle() {

}

func (s *AppService) NewID() string {
	return strconv.FormatInt(s.IID.NewID(), 36)
}

func (s *AppService) NewSecret() string {
	m := md5.New()
	m.Write([]byte(uuid.New().String()))
	return hex.EncodeToString(m.Sum(nil))
}

func GetAppService(ctx micro.Context, name string) (*AppService, error) {
	s, err := ctx.GetService(name)
	if err != nil {
		return nil, err
	}
	ss, ok := s.(*AppService)
	if ok {
		return ss, nil
	}
	return nil, fmt.Errorf("service %s not instanceof *AppService", name)
}
