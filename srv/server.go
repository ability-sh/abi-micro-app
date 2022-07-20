package srv

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ability-sh/abi-lib/dynamic"
	"github.com/ability-sh/abi-lib/json"
	"github.com/ability-sh/abi-micro-app/pb"
	"github.com/ability-sh/abi-micro/grpc"
	"github.com/ability-sh/abi-micro/mongodb"
	"github.com/ability-sh/abi-micro/oss"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	G "google.golang.org/grpc"
)

type server struct {
}

func encodeObject(v interface{}) string {
	if v == nil {
		return ""
	}
	r, _ := json.Marshal(v)
	return string(r)
}

func decodeObject(s string) interface{} {
	if s == "" {
		return nil
	}
	var v interface{} = nil
	json.Unmarshal([]byte(s), &v)
	return v
}

func setApp(a *pb.App, rs bson.M) {
	a.Id = dynamic.StringValue(rs["_id"], "")
	a.Ctime = int32(dynamic.IntValue(rs["ctime"], 0))
	a.Secret = dynamic.StringValue(rs["secret"], "")
	a.Title = dynamic.StringValue(rs["title"], "")
	a.Info = encodeObject(rs["info"])
}

func toAppItems(rs []bson.M) []*pb.App {
	vs := []*pb.App{}
	for _, r := range rs {
		v := &pb.App{}
		setApp(v, r)
		vs = append(vs, v)
	}
	return vs
}

func setVer(a *pb.Ver, rs bson.M) {
	a.Appid = dynamic.StringValue(rs["appid"], "")
	a.Ver = dynamic.StringValue(rs["ver"], "")
	a.Ctime = int32(dynamic.IntValue(rs["ctime"], 0))
	a.Title = dynamic.StringValue(rs["title"], "")
	a.Info = encodeObject(rs["info"])
	a.Status = int32(dynamic.IntValue(rs["status"], 0))
}

func toVerItems(rs []bson.M) []*pb.Ver {
	vs := []*pb.Ver{}
	for _, r := range rs {
		v := &pb.Ver{}
		setVer(v, r)
		vs = append(vs, v)
	}
	return vs
}

func setContainer(a *pb.Container, rs bson.M) {
	a.Id = dynamic.StringValue(rs["_id"], "")
	a.Ctime = int32(dynamic.IntValue(rs["ctime"], 0))
	a.Secret = dynamic.StringValue(rs["secret"], "")
	a.Title = dynamic.StringValue(rs["title"], "")
	a.Info = encodeObject(rs["info"])
	a.Env = map[string]string{}
	dynamic.Each(rs["env"], func(key interface{}, value interface{}) bool {
		a.Env[dynamic.StringValue(key, "")] = dynamic.StringValue(value, "")
		return true
	})
}

func toContainerItems(rs []bson.M) []*pb.Container {
	vs := []*pb.Container{}
	for _, r := range rs {
		v := &pb.Container{}
		setContainer(v, r)
		vs = append(vs, v)
	}
	return vs
}

func setAc(a *pb.Ac, rs bson.M) {
	a.Cid = dynamic.StringValue(rs["cid"], "")
	a.Appid = dynamic.StringValue(rs["appid"], "")
	a.Ctime = int32(dynamic.IntValue(rs["ctime"], 0))
	a.Ver = dynamic.StringValue(rs["ver"], "")
	a.Title = dynamic.StringValue(rs["title"], "")
	a.Info = encodeObject(rs["info"])
	a.Env = map[string]string{}
	dynamic.Each(rs["env"], func(key interface{}, value interface{}) bool {
		a.Env[dynamic.StringValue(key, "")] = dynamic.StringValue(value, "")
		return true
	})
}

func toAcItems(rs []bson.M) []*pb.Ac {
	vs := []*pb.Ac{}
	for _, r := range rs {
		v := &pb.Ac{}
		setAc(v, r)
		vs = append(vs, v)
	}
	return vs
}

func (s *server) AppCreate(c context.Context, task *pb.AppCreateTask) (*pb.AppResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	var info interface{} = nil

	if task.Info != "" {
		err := json.Unmarshal([]byte(task.Info), &info)
		if err != nil {
			return &pb.AppResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.AppResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.AppResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(app.Db)

	db_app := db.Collection("app")

	id := app.NewID()
	secret := app.NewSecret()
	ctime := int32(time.Now().Unix())

	_, err = db_app.InsertOne(c,
		bson.D{bson.E{"_id", id},
			bson.E{"title", task.Title},
			bson.E{"info", info},
			bson.E{"secret", secret},
			bson.E{"ctime", ctime}})

	if err != nil {
		return &pb.AppResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	a := &pb.App{Id: id, Title: task.Title, Info: task.Info, Secret: secret, Ctime: ctime}

	return &pb.AppResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) AppRemove(c context.Context, task *pb.AppRemoveTask) (*pb.AppResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Appid == "" {
		return &pb.AppResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param appid"}, nil
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.AppResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.AppResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(app.Db)

	db_app := db.Collection("app")

	var rs bson.M

	err = db_app.FindOneAndDelete(c,
		bson.D{bson.E{"_id", task.Appid}}).Decode(&rs)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &pb.AppResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found app"}, nil
		}
		return &pb.AppResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	a := &pb.App{}

	setApp(a, rs)

	return &pb.AppResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) AppSet(c context.Context, task *pb.AppSetTask) (*pb.AppResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Appid == "" {
		return &pb.AppResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param appid"}, nil
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.AppResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.AppResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(app.Db)

	db_app := db.Collection("app")

	set := bson.D{}

	if task.Title != "" {
		set = append(set, bson.E{"title", task.Title})
	}

	secret := ""

	if task.Secret {
		secret = app.NewSecret()
		set = append(set, bson.E{"secret", secret})
	}

	var info interface{} = nil

	if task.Info != "" {
		err = json.Unmarshal([]byte(task.Info), &info)
		if err != nil {
			return &pb.AppResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}
		dynamic.Each(info, func(key interface{}, value interface{}) bool {
			set = append(set, bson.E{fmt.Sprintf("info.%s", dynamic.StringValue(key, "")), value})
			return true
		})
	}

	a := &pb.App{}

	if len(set) == 0 {

		var rs bson.M

		err = db_app.FindOne(c,
			bson.D{bson.E{"_id", task.Appid}}).Decode(&rs)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return &pb.AppResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found app"}, nil
			}
			return &pb.AppResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}

		setApp(a, rs)

	} else {

		opts := options.FindOneAndUpdate().SetUpsert(true)

		var rs bson.M

		err = db_app.FindOneAndUpdate(c,
			bson.D{bson.E{"_id", task.Appid}}, bson.D{bson.E{"$set", set}}, opts).Decode(&rs)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return &pb.AppResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found app"}, nil
			}
			return &pb.AppResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}

		{
			var i = rs["info"]
			if i == nil {
				i = map[string]interface{}{}
				rs["info"] = i
			}
			dynamic.Each(info, func(key interface{}, value interface{}) bool {
				dynamic.Set(i, dynamic.StringValue(key, ""), value)
				return true
			})
		}

		if task.Title != "" {
			rs["title"] = task.Title
		}

		if task.Secret {
			rs["title"] = secret
		}

		setApp(a, rs)

	}

	return &pb.AppResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) AppGet(c context.Context, task *pb.AppGetTask) (*pb.AppResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Appid == "" {
		return &pb.AppResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param appid"}, nil
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.AppResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.AppResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(app.Db)

	db_app := db.Collection("app")

	a := &pb.App{}

	var rs bson.M

	err = db_app.FindOne(c,
		bson.D{bson.E{"_id", task.Appid}}).Decode(&rs)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &pb.AppResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found app"}, nil
		}
		return &pb.AppResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	setApp(a, rs)

	return &pb.AppResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) AppQuery(c context.Context, task *pb.AppQueryTask) (*pb.AppQueryResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.AppQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.AppQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	n := task.N
	p := task.P

	if n < 1 {
		n = 20
	}

	rs := &pb.AppQueryResult{}

	db := conn.Database(app.Db)

	db_app := db.Collection("app")

	opts := options.Find().SetSort(bson.D{bson.E{"ctime", -1}}).SetLimit(int64(n))

	filter := bson.D{}

	if task.Q != "" {
		filter = append(filter, bson.E{"title", bson.E{"$regex", task.Q}})
	}

	if p > 0 {

		opts = opts.SetSkip(int64(n * (p - 1)))

		totalCount, err := db_app.CountDocuments(c, filter)

		if err != nil {
			return &pb.AppQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}

		count := int32(totalCount) / n

		if int32(totalCount)%n != 0 {
			count = count + 1
		}

		rs.Page = &pb.Page{P: p, N: n, TotalCount: int32(totalCount), Count: count}

	}

	cursor, err := db_app.Find(c, filter, opts)

	if err != nil {
		return &pb.AppQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	defer cursor.Close(c)

	var items []bson.M

	err = cursor.All(c, &items)

	if err != nil {
		return &pb.AppQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	rs.Items = toAppItems(items)
	rs.Errno = ERRNO_OK

	return rs, nil
}

func (s *server) VerCreate(c context.Context, task *pb.VerCreateTask) (*pb.VerResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Appid == "" {
		return &pb.VerResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param appid"}, nil
	}

	if task.Ver == "" {
		return &pb.VerResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param ver"}, nil
	}

	var info interface{} = nil

	if task.Info != "" {
		err := json.Unmarshal([]byte(task.Info), &info)
		if err != nil {
			return &pb.VerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.VerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.VerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(app.Db)

	db_ver := db.Collection("ver")

	ctime := int32(time.Now().Unix())

	_, err = db_ver.InsertOne(c,
		bson.D{bson.E{"title", task.Title},
			bson.E{"info", info},
			bson.E{"appid", task.Appid},
			bson.E{"ver", task.Ver},
			bson.E{"status", task.Status},
			bson.E{"ctime", ctime}})

	if err != nil {
		return &pb.VerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	a := &pb.Ver{Title: task.Title, Info: task.Info, Appid: task.Appid, Ver: task.Ver, Ctime: ctime}

	return &pb.VerResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) VerRemove(c context.Context, task *pb.VerRemoveTask) (*pb.VerResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Appid == "" {
		return &pb.VerResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param appid"}, nil
	}

	if task.Ver == "" {
		return &pb.VerResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param ver"}, nil
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.VerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.VerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(app.Db)

	db_ver := db.Collection("ver")

	var rs bson.M

	err = db_ver.FindOneAndDelete(c,
		bson.D{bson.E{"appid", task.Appid}, bson.E{"ver", task.Ver}}).Decode(&rs)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &pb.VerResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found app"}, nil
		}
		return &pb.VerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	a := &pb.Ver{}

	setVer(a, rs)

	return &pb.VerResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) VerSet(c context.Context, task *pb.VerSetTask) (*pb.VerResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	ctx.Println("task", task)

	if task.Appid == "" {
		return &pb.VerResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param appid"}, nil
	}

	if task.Ver == "" {
		return &pb.VerResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param ver"}, nil
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.VerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.VerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(app.Db)

	db_ver := db.Collection("ver")

	set := bson.D{}

	if task.Title != "" {
		set = append(set, bson.E{"title", task.Title})
	}

	if task.Status != "" {
		s, _ := strconv.Atoi(task.Status)
		set = append(set, bson.E{"status", s})
	}

	var info interface{} = nil

	if task.Info != "" {
		err = json.Unmarshal([]byte(task.Info), &info)
		if err != nil {
			return &pb.VerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}
		dynamic.Each(info, func(key interface{}, value interface{}) bool {
			set = append(set, bson.E{fmt.Sprintf("info.%s", dynamic.StringValue(key, "")), value})
			return true
		})
	}

	a := &pb.Ver{}

	if len(set) == 0 {

		var rs bson.M

		err = db_ver.FindOne(c,
			bson.D{bson.E{"appid", task.Appid}, bson.E{"ver", task.Ver}}).Decode(&rs)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return &pb.VerResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found app"}, nil
			}
			return &pb.VerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}

		setVer(a, rs)

	} else {

		opts := options.FindOneAndUpdate().SetUpsert(true)

		var rs bson.M

		err = db_ver.FindOneAndUpdate(c,
			bson.D{bson.E{"appid", task.Appid}, bson.E{"ver", task.Ver}}, bson.D{bson.E{"$set", set}}, opts).Decode(&rs)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return &pb.VerResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found ver"}, nil
			}
			return &pb.VerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}

		{
			var i = rs["info"]
			if i == nil {
				i = map[string]interface{}{}
				rs["info"] = i
			}
			dynamic.Each(info, func(key interface{}, value interface{}) bool {
				dynamic.Set(i, dynamic.StringValue(key, ""), value)
				return true
			})
		}

		if task.Title != "" {
			rs["title"] = task.Title
		}

		if task.Status != "" {
			rs["status"], _ = strconv.Atoi(task.Status)
		}

		setVer(a, rs)

	}

	return &pb.VerResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) VerGet(c context.Context, task *pb.VerGetTask) (*pb.VerResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Appid == "" {
		return &pb.VerResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param appid"}, nil
	}

	if task.Ver == "" {
		return &pb.VerResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param ver"}, nil
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.VerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.VerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(app.Db)

	db_ver := db.Collection("ver")

	a := &pb.Ver{}

	var rs bson.M

	err = db_ver.FindOne(c,
		bson.D{bson.E{"appid", task.Appid}, bson.E{"ver", task.Ver}}).Decode(&rs)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &pb.VerResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found app"}, nil
		}
		return &pb.VerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	setVer(a, rs)

	return &pb.VerResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) VerQuery(c context.Context, task *pb.VerQueryTask) (*pb.VerQueryResult, error) {
	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Appid == "" {
		return &pb.VerQueryResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param appid"}, nil
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.VerQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.VerQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	n := task.N
	p := task.P

	if n < 1 {
		n = 20
	}

	rs := &pb.VerQueryResult{}

	db := conn.Database(app.Db)

	db_ver := db.Collection("ver")

	opts := options.Find().SetSort(bson.D{bson.E{"ctime", -1}}).SetLimit(int64(n))

	filter := bson.D{bson.E{"appid", task.Appid}}

	if task.Q != "" {
		filter = append(filter, bson.E{"title", bson.E{"$regex", task.Q}})
	}

	if task.Status != "" {
		vs := bson.D{}
		for _, s := range strings.Split(task.Status, ",") {
			ss, _ := strconv.Atoi(s)
			vs = append(vs, bson.E{"status", ss})
		}
		filter = append(filter, bson.E{"$or", vs})
	}

	if p > 0 {

		opts = opts.SetSkip(int64(n * (p - 1)))

		totalCount, err := db_ver.CountDocuments(c, filter)

		if err != nil {
			return &pb.VerQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}

		count := int32(totalCount) / n

		if int32(totalCount)%n != 0 {
			count = count + 1
		}

		rs.Page = &pb.Page{P: p, N: n, TotalCount: int32(totalCount), Count: count}

	}

	cursor, err := db_ver.Find(c, filter, opts)

	if err != nil {
		return &pb.VerQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	defer cursor.Close(c)

	var items []bson.M

	err = cursor.All(c, &items)

	if err != nil {
		return &pb.VerQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	rs.Items = toVerItems(items)
	rs.Errno = ERRNO_OK

	return rs, nil
}

func (s *server) VerGetURL(c context.Context, task *pb.VerGetURLTask) (*pb.VerGetURLResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Appid == "" {
		return &pb.VerGetURLResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param appid"}, nil
	}

	if task.Ver == "" {
		return &pb.VerGetURLResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param ver"}, nil
	}

	if task.Ability == "" {
		return &pb.VerGetURLResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param ability"}, nil
	}

	if task.Expires <= 0 {
		task.Expires = 300
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.VerGetURLResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	ss, err := oss.GetOSS(ctx, SERVICE_OSS)

	if err != nil {
		return &pb.VerGetURLResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	u, err := ss.GetSignURL(fmt.Sprintf("%s%s/v%s-%s.zip", app.BasePath, task.Appid, task.Ver, task.Ability), time.Duration(task.Expires)*time.Second)

	if err != nil {
		return &pb.VerGetURLResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	return &pb.VerGetURLResult{Errno: ERRNO_OK, Data: u}, nil

}

func (s *server) VerUpURL(c context.Context, task *pb.VerUpURLTask) (*pb.VerUpURLResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Appid == "" {
		return &pb.VerUpURLResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param appid"}, nil
	}

	if task.Ver == "" {
		return &pb.VerUpURLResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param ver"}, nil
	}

	if task.Ability == "" {
		return &pb.VerUpURLResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param ability"}, nil
	}

	if task.Expires <= 0 {
		task.Expires = 300
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.VerUpURLResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	ss, err := oss.GetOSS(ctx, SERVICE_OSS)

	if err != nil {
		return &pb.VerUpURLResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	u, data, err := ss.PostSignURL(fmt.Sprintf("%s%s/v%s-%s.zip", app.BasePath, task.Appid, task.Ver, task.Ability), time.Duration(task.Expires)*time.Second, app.AppMaxSize, nil)

	if err != nil {
		return &pb.VerUpURLResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	return &pb.VerUpURLResult{Errno: ERRNO_OK, Data: &pb.VerUpURL{Url: u, Data: data, Method: "POST", Key: "file"}}, nil

}

func (s *server) ContainerCreate(c context.Context, task *pb.ContainerCreateTask) (*pb.ContainerResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	var info interface{} = nil

	if task.Info != "" {
		err := json.Unmarshal([]byte(task.Info), &info)
		if err != nil {
			return &pb.ContainerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.ContainerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.ContainerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(app.Db)

	db_container := db.Collection("container")

	id := app.NewID()
	ctime := int32(time.Now().Unix())
	secret := app.NewSecret()

	_, err = db_container.InsertOne(c,
		bson.D{bson.E{"_id", id},
			bson.E{"title", task.Title},
			bson.E{"info", info},
			bson.E{"env", task.Env},
			bson.E{"secret", secret},
			bson.E{"ctime", ctime}})

	if err != nil {
		return &pb.ContainerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	a := &pb.Container{Id: id, Title: task.Title, Info: task.Info, Env: task.Env, Ctime: ctime, Secret: secret}

	return &pb.ContainerResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) ContainerRemove(c context.Context, task *pb.ContainerRemoveTask) (*pb.ContainerResult, error) {
	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Cid == "" {
		return &pb.ContainerResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param cid"}, nil
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.ContainerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.ContainerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(app.Db)

	db_container := db.Collection("container")

	var rs bson.M

	err = db_container.FindOneAndDelete(c,
		bson.D{bson.E{"_id", task.Cid}}).Decode(&rs)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &pb.ContainerResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found app"}, nil
		}
		return &pb.ContainerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	a := &pb.Container{}

	setContainer(a, rs)

	return &pb.ContainerResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) ContainerSet(c context.Context, task *pb.ContainerSetTask) (*pb.ContainerResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Cid == "" {
		return &pb.ContainerResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param cid"}, nil
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.ContainerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.ContainerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(app.Db)

	db_container := db.Collection("container")

	set := bson.D{}

	if task.Title != "" {
		set = append(set, bson.E{"title", task.Title})
	}

	secret := ""

	if task.Secret {
		secret = app.NewSecret()
		set = append(set, bson.E{"secret", secret})
	}

	var info interface{} = nil

	if task.Info != "" {
		err = json.Unmarshal([]byte(task.Info), &info)
		if err != nil {
			return &pb.ContainerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}
		dynamic.Each(info, func(key interface{}, value interface{}) bool {
			set = append(set, bson.E{fmt.Sprintf("info.%s", dynamic.StringValue(key, "")), value})
			return true
		})
	}

	if task.Env != nil {
		dynamic.Each(task.Env, func(key interface{}, value interface{}) bool {
			set = append(set, bson.E{fmt.Sprintf("env.%s", dynamic.StringValue(key, "")), value})
			return true
		})
	}

	a := &pb.Container{}

	if len(set) == 0 {

		var rs bson.M

		err = db_container.FindOne(c,
			bson.D{bson.E{"_id", task.Cid}}).Decode(&rs)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return &pb.ContainerResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found container"}, nil
			}
			return &pb.ContainerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}

		setContainer(a, rs)

	} else {

		opts := options.FindOneAndUpdate().SetUpsert(true)

		var rs bson.M

		err = db_container.FindOneAndUpdate(c,
			bson.D{bson.E{"_id", task.Cid}}, bson.D{bson.E{"$set", set}}, opts).Decode(&rs)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return &pb.ContainerResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found container"}, nil
			}
			return &pb.ContainerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}

		{
			var i = rs["info"]
			if i == nil {
				i = map[string]interface{}{}
				rs["info"] = i
			}
			dynamic.Each(info, func(key interface{}, value interface{}) bool {
				dynamic.Set(i, dynamic.StringValue(key, ""), value)
				return true
			})
		}

		{
			var i = rs["env"]
			if i == nil {
				i = map[string]string{}
				rs["env"] = i
			}
			dynamic.Each(task.Env, func(key interface{}, value interface{}) bool {
				dynamic.Set(i, dynamic.StringValue(key, ""), value)
				return true
			})
		}

		if task.Title != "" {
			rs["title"] = task.Title
		}

		if task.Secret {
			rs["title"] = secret
		}

		setContainer(a, rs)

	}

	return &pb.ContainerResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) ContainerGet(c context.Context, task *pb.ContainerGetTask) (*pb.ContainerResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Cid == "" {
		return &pb.ContainerResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param cid"}, nil
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.ContainerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.ContainerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(app.Db)

	db_container := db.Collection("container")

	a := &pb.Container{}

	var rs bson.M

	err = db_container.FindOne(c,
		bson.D{bson.E{"_id", task.Cid}}).Decode(&rs)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &pb.ContainerResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found container"}, nil
		}
		return &pb.ContainerResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	setContainer(a, rs)

	return &pb.ContainerResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) ContainerQuery(c context.Context, task *pb.ContainerQueryTask) (*pb.ContainerQueryResult, error) {
	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.ContainerQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.ContainerQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	n := task.N
	p := task.P

	if n < 1 {
		n = 20
	}

	rs := &pb.ContainerQueryResult{}

	db := conn.Database(app.Db)

	db_container := db.Collection("container")

	opts := options.Find().SetSort(bson.D{bson.E{"ctime", -1}}).SetLimit(int64(n))

	filter := bson.D{}

	if task.Q != "" {
		filter = append(filter, bson.E{"title", bson.E{"$regex", task.Q}})
	}

	if p > 0 {

		opts = opts.SetSkip(int64(n * (p - 1)))

		totalCount, err := db_container.CountDocuments(c, filter)

		if err != nil {
			return &pb.ContainerQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}

		count := int32(totalCount) / n

		if int32(totalCount)%n != 0 {
			count = count + 1
		}

		rs.Page = &pb.Page{P: p, N: n, TotalCount: int32(totalCount), Count: count}

	}

	cursor, err := db_container.Find(c, filter, opts)

	if err != nil {
		return &pb.ContainerQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	defer cursor.Close(c)

	var items []bson.M

	err = cursor.All(c, &items)

	if err != nil {
		return &pb.ContainerQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	rs.Items = toContainerItems(items)
	rs.Errno = ERRNO_OK

	return rs, nil
}

func (s *server) AcAdd(c context.Context, task *pb.AcAddTask) (*pb.AcResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Cid == "" {
		return &pb.AcResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param cid"}, nil
	}

	if task.Appid == "" {
		return &pb.AcResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param appid"}, nil
	}

	var info interface{} = nil

	if task.Info != "" {
		err := json.Unmarshal([]byte(task.Info), &info)
		if err != nil {
			return &pb.AcResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.AcResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.AcResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(app.Db)

	db_ac := db.Collection("ac")

	ctime := int32(time.Now().Unix())

	_, err = db_ac.InsertOne(c,
		bson.D{bson.E{"cid", task.Cid},
			bson.E{"appid", task.Appid},
			bson.E{"title", task.Title},
			bson.E{"info", info},
			bson.E{"env", task.Env},
			bson.E{"ver", task.Ver},
			bson.E{"ctime", ctime}})

	if err != nil {
		return &pb.AcResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	a := &pb.Ac{Cid: task.Cid, Appid: task.Appid, Title: task.Title, Info: task.Info, Env: task.Env, Ver: task.Ver, Ctime: ctime}

	return &pb.AcResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) AcRemove(c context.Context, task *pb.AcRemoveTask) (*pb.AcResult, error) {
	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Cid == "" {
		return &pb.AcResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param cid"}, nil
	}

	if task.Appid == "" {
		return &pb.AcResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param appid"}, nil
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.AcResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.AcResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(app.Db)

	db_ac := db.Collection("ac")

	var rs bson.M

	err = db_ac.FindOneAndDelete(c,
		bson.D{bson.E{"cid", task.Cid}, bson.E{"appid", task.Appid}}).Decode(&rs)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &pb.AcResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found app"}, nil
		}
		return &pb.AcResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	a := &pb.Ac{}

	setAc(a, rs)

	return &pb.AcResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) AcSet(c context.Context, task *pb.AcSetTask) (*pb.AcResult, error) {
	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Cid == "" {
		return &pb.AcResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param cid"}, nil
	}

	if task.Appid == "" {
		return &pb.AcResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param appid"}, nil
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.AcResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.AcResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(app.Db)

	db_ac := db.Collection("ac")

	set := bson.D{}

	if task.Title != "" {
		set = append(set, bson.E{"title", task.Title})
	}

	var info interface{} = nil

	if task.Info != "" {
		err = json.Unmarshal([]byte(task.Info), &info)
		if err != nil {
			return &pb.AcResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}
		dynamic.Each(info, func(key interface{}, value interface{}) bool {
			set = append(set, bson.E{fmt.Sprintf("info.%s", dynamic.StringValue(key, "")), value})
			return true
		})
	}

	if task.Env != nil {
		dynamic.Each(task.Env, func(key interface{}, value interface{}) bool {
			set = append(set, bson.E{fmt.Sprintf("env.%s", dynamic.StringValue(key, "")), value})
			return true
		})
	}

	if task.Ver != "" {
		set = append(set, bson.E{"ver", task.Ver})
	}

	a := &pb.Ac{}

	if len(set) == 0 {

		var rs bson.M

		err = db_ac.FindOne(c,
			bson.D{bson.E{"cid", task.Cid}, bson.E{"appid", task.Appid}}).Decode(&rs)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return &pb.AcResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found app"}, nil
			}
			return &pb.AcResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}

		setAc(a, rs)

	} else {

		opts := options.FindOneAndUpdate().SetUpsert(true)

		var rs bson.M

		err = db_ac.FindOneAndUpdate(c,
			bson.D{bson.E{"cid", task.Cid}, bson.E{"appid", task.Appid}}, bson.D{bson.E{"$set", set}}, opts).Decode(&rs)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return &pb.AcResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found app"}, nil
			}
			return &pb.AcResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}

		{
			var i = rs["info"]
			if i == nil {
				i = map[string]interface{}{}
				rs["info"] = i
			}
			dynamic.Each(info, func(key interface{}, value interface{}) bool {
				dynamic.Set(i, dynamic.StringValue(key, ""), value)
				return true
			})
		}

		{
			var i = rs["env"]
			if i == nil {
				i = map[string]string{}
				rs["env"] = i
			}
			dynamic.Each(task.Env, func(key interface{}, value interface{}) bool {
				dynamic.Set(i, dynamic.StringValue(key, ""), value)
				return true
			})
		}

		if task.Title != "" {
			rs["title"] = task.Title
		}

		if task.Ver != "" {
			rs["ver"] = task.Ver
		}

		setAc(a, rs)

	}

	return &pb.AcResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) AcGet(c context.Context, task *pb.AcGetTask) (*pb.AcResult, error) {
	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Cid == "" {
		return &pb.AcResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param cid"}, nil
	}

	if task.Appid == "" {
		return &pb.AcResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param appid"}, nil
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.AcResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.AcResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	db := conn.Database(app.Db)

	db_ac := db.Collection("ac")

	a := &pb.Ac{}

	var rs bson.M

	err = db_ac.FindOne(c,
		bson.D{bson.E{"cid", task.Cid}, bson.E{"Appid", task.Appid}}).Decode(&rs)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &pb.AcResult{Errno: ERRNO_NOT_FOUND, Errmsg: "not found ac"}, nil
		}
		return &pb.AcResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	setAc(a, rs)

	return &pb.AcResult{Errno: ERRNO_OK, Data: a}, nil
}

func (s *server) AcQuery(c context.Context, task *pb.AcQueryTask) (*pb.AcQueryResult, error) {

	ctx := grpc.GetContext(c)

	defer ctx.Recycle()

	if task.Cid == "" && task.Appid == "" {
		return &pb.AcQueryResult{Errno: ERRNO_INPUT_DATA, Errmsg: "not found param cid or appid"}, nil
	}

	app, err := GetAppService(ctx, SERVICE_APP)

	if err != nil {
		return &pb.AcQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	conn, err := mongodb.GetClient(ctx, SERVICE_MONGODB)

	if err != nil {
		return &pb.AcQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	n := task.N
	p := task.P

	if n < 1 {
		n = 20
	}

	rs := &pb.AcQueryResult{}

	db := conn.Database(app.Db)

	db_ac := db.Collection("ac")

	opts := options.Find().SetSort(bson.D{bson.E{"ctime", -1}}).SetLimit(int64(n))

	filter := bson.D{}

	if task.Cid != "" {
		filter = append(filter, bson.E{"cid", task.Cid})
	}

	if task.Appid != "" {
		filter = append(filter, bson.E{"appid", task.Appid})
	}

	if task.Q != "" {
		filter = append(filter, bson.E{"title", bson.E{"$regex", task.Q}})
	}

	if p > 0 {

		opts = opts.SetSkip(int64(n * (p - 1)))

		totalCount, err := db_ac.CountDocuments(c, filter)

		if err != nil {
			return &pb.AcQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
		}

		count := int32(totalCount) / n

		if int32(totalCount)%n != 0 {
			count = count + 1
		}

		rs.Page = &pb.Page{P: p, N: n, TotalCount: int32(totalCount), Count: count}

	}

	cursor, err := db_ac.Find(c, filter, opts)

	if err != nil {
		return &pb.AcQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	defer cursor.Close(c)

	var items []bson.M

	err = cursor.All(c, &items)

	if err != nil {
		return &pb.AcQueryResult{Errno: ERRNO_INTERNAL_SERVER, Errmsg: err.Error()}, nil
	}

	rs.Items = toAcItems(items)
	rs.Errno = ERRNO_OK

	return rs, nil
}

func Reg(s *G.Server) {
	pb.RegisterServiceServer(s, &server{})
}
