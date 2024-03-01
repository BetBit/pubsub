package pubsub

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Auth struct {
	clientId   string
	projectId  string
	token      string
	events     []string
	subscribes []string
}

func (a *Auth) WithContext(ctx context.Context) context.Context {
	values := url.Values{}

	clientId := a.clientId
	values.Set("client_id", clientId)

	projectId := a.projectId
	values.Set("project_id", projectId)

	nonce := getMD5Encode(uuid.New().String())
	values.Set("nonce", nonce)

	timestamp := strconv.Itoa(int(time.Now().Unix()))
	values.Set("timestamp", timestamp)

	xSign := values.Encode()
	sign := a.createHash(xSign)

	events := strings.Join(a.events, ",")
	subscribes := strings.Join(a.subscribes, ",")

	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"client-id", clientId,
		"project-id", projectId,
		"nonce", nonce,
		"timestamp", timestamp,
		"sign", sign,
		"events", events,
		"subscribes", subscribes,
	))
}

func (a *Auth) createHash(s string) string {
	token := []byte(a.token)
	hm := hmac.New(sha1.New, token)
	hm.Write([]byte(s))
	return hex.EncodeToString(hm.Sum(nil))
}

func getMD5Encode(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
