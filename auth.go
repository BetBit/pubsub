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
	"time"
)

type Auth struct {
	clientId string
	agentId  string
	token    string
}

func (a *Auth) WithContext(ctx context.Context) context.Context {
	values := url.Values{}

	clientId := a.clientId
	values.Set("client_id", clientId)

	agentId := a.agentId
	values.Set("agent_id", agentId)

	nonce := getMD5Encode(uuid.New().String())
	values.Set("nonce", nonce)

	timestamp := strconv.Itoa(int(time.Now().Unix()))
	values.Set("timestamp", timestamp)

	xSign := values.Encode()
	sign := a.createHash(xSign)

	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"client-id", clientId,
		"agent-id", agentId,
		"nonce", nonce,
		"timestamp", timestamp,
		"sign", sign,
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
