package driver

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	MONGODB_DRIVER = "mongodb"
)

type MongodbOpts struct {
	DSN           string
	Scheme        string
	Username      string
	Password      string
	Port          int
	Hosts         []string
	Timeout       int
	Database      string
	TLS           bool
	TLSInsecure   bool
	AuthSource    string
	AuthMechanism string
	ReplicaSet    string
	Collection    string
	Document      string
	Create        bool
	Logger        *slog.Logger
}

type Mongodb struct {
	client *mongo.Client
	uri    *url.URL
	opts   MongodbOpts
	logger *slog.Logger
}

type MongodbResult struct {
	ID int                 `bson:"id"`
	Ts primitive.Timestamp `bson:"ts"`
}

func NewMongodb(opts MongodbOpts) (m *Mongodb, err error) {
	if opts.Timeout == 0 {
		opts.Timeout = TIMEOUT
	}

	if opts.Database == "" {
		return nil, fmt.Errorf("database name is required")
	}

	if opts.Collection == "" {
		return nil, fmt.Errorf("collection name is required")
	}

	var logger *slog.Logger
	if opts.Logger != nil {
		logger = opts.Logger.With("driver", MONGODB_DRIVER)
	} else {
		logger = slog.With("driver", MONGODB_DRIVER)
	}

	m = &Mongodb{
		opts:   opts,
		logger: logger,
	}

	m.uri, err = m.parseURI()
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m *Mongodb) parseURI() (*url.URL, error) {
	if m.opts.DSN != "" {
		return url.Parse(m.opts.DSN)
	}

	var uri string
	if m.opts.Scheme != "" {
		uri = m.opts.Scheme
	} else {
		uri = "mongodb"
	}
	if !strings.HasSuffix(uri, "://") {
		uri = uri + "://"
	}

	if len(m.opts.Hosts) > 0 {
		uri = uri + strings.Join(m.opts.Hosts, ",")
	} else {
		return nil, fmt.Errorf("invalid mongodb hosts")
	}

	uri = uri + "/" + m.opts.Database

	if m.opts.ReplicaSet != "" {
		uri = uri + m.opts.ReplicaSet
	}

	url, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	queryParams := url.Query()

	if m.opts.TLS {
		queryParams.Add("tls", "true")
		if m.opts.TLSInsecure {
			queryParams.Add("tlsInsecure", "true")
		}
	}

	url.RawQuery = queryParams.Encode()

	return url, nil
}

func (m *Mongodb) Connect() (err error) {
	m.logger.Debug("connecting")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.opts.Timeout)*time.Second)
	defer cancel()

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	credentials := options.Credential{
		AuthMechanism: m.opts.AuthMechanism,
		AuthSource:    m.opts.AuthSource,
		Username:      m.opts.Username,
		Password:      m.opts.Password,
	}
	co := options.Client().ApplyURI(m.uri.String()).SetServerAPIOptions(serverAPI).SetAuth(credentials)

	m.client, err = mongo.Connect(context.Background(), co)
	if err != nil {
		return err
	}

	m.logger.Debug("sending ping")

	if err = m.client.Ping(ctx, readpref.Primary()); err != nil {
		return err
	}

	m.logger.Debug("connected")
	return nil
}

func (m *Mongodb) Read() error {
	m.logger.Debug("reading")

	var result *MongodbResult

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.opts.Timeout)*time.Second)
	defer cancel()

	collection := m.client.Database(m.opts.Database).Collection(m.opts.Collection)
	if err := collection.FindOne(ctx, bson.M{"id": 1}).Decode(&result); err != nil {
		if err.Error() == "mongo: no documents in result" && m.opts.Create {
			m.logger.Debug("creating initial document")
			if err = m.Write(); err != nil {
				return err
			}
		}
		return err
	}

	m.logger.Debug("read", slog.Any("result", result))
	return nil
}

func (m *Mongodb) Write() error {
	m.logger.Debug("writing")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.opts.Timeout)*time.Second)
	defer cancel()

	collection := m.client.Database(m.opts.Database).Collection(m.opts.Collection)

	filter := bson.M{"id": 1}
	ts := primitive.Timestamp{T: uint32(time.Now().Unix())}
	update := bson.M{"$set": bson.M{"id": 1, "ts": ts}}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}

	m.logger.Debug("written")
	return nil
}

func (m *Mongodb) Disconnect() error {
	if m.client != nil {
		m.logger.Debug("disconnecting")
		err := m.client.Disconnect(context.Background())
		if err != nil {
			return err
		}
		m.logger.Debug("disconnected")
	}
	return nil
}
