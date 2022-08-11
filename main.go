package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"golang.org/x/net/context"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/client"
)

type ststoken struct {
	AccessKeyId     string
	SecretAccessKey string
	Token           string
	Code            string
	Type            string
	LastUpdated     string
	Expiration      string
}

type SimpleContainer struct {
	Config *SimpleContainerConfig
	Name   string
}

type SimpleContainerConfig struct {
	Env []string
}

type CredentialStore struct {
	credsMap    map[string]*credentials.Credentials
	sess        *session.Session
	defaultRole string
}

type Env struct {
	credentialStore *CredentialStore
	region          string
}

func NewCredentialStore(defaultRole string) (store *CredentialStore) {
	sess := session.Must(session.NewSession())
	credsMap := make(map[string]*credentials.Credentials)

	store = &CredentialStore{sess: sess,
		credsMap:    credsMap,
		defaultRole: defaultRole}

	return store
}

var credentialStore *CredentialStore

func main() {

	region := flag.String("r", "us-east-1", "AWS region")
	showHelp := flag.Bool("h", false, "Show help")
	flag.Parse()

	if *showHelp {
		fmt.Println("Usage: metadata [default role arn]")
		os.Exit(0)
	}

	defaultRole := os.Getenv("DEFAULT_IAM_ROLE")

	if len(flag.Args()) > 0 {
		defaultRole = flag.Args()[0]
	}

	if defaultRole != "" {
		fmt.Printf("Using %s as default role\n", defaultRole)
	}

	credentialStore = NewCredentialStore(defaultRole)
	env := &Env{credentialStore: credentialStore, region: *region}

	mux := http.NewServeMux()
	// Currently we handle only credentials
	mux.HandleFunc("/", env.credHandler)

	s := &http.Server{
		Addr:         "169.254.169.254:80",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	log.Fatal(s.ListenAndServe())
}

func (env *Env) credHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)

	if err != nil {
		log.Printf("ERROR: Failed to parse: %s: %s\n", r.RemoteAddr, err)
	}

	container, err := containerFromIp(ip)

	if err != nil {
		http.Error(w, fmt.Sprintf("Internal Error: %s", err), http.StatusInternalServerError)
		return
	}

	role, ok := container.getEnvValue("IAM_ROLE")

	if !ok {
		if env.credentialStore.defaultRole != "" {
			role = env.credentialStore.defaultRole
			log.Printf("No IAM_ROLE found for %s (%s), using default role (%s)\n", ip, container.Name, role)
		} else {
			log.Printf("ERROR: No IAM_ROLE found for %s (%s)\n", ip, container.Name)

			http.Error(w, fmt.Sprintf("Internal Error: %s", err), http.StatusInternalServerError)
			return
		}
	}

	log.Printf("GET: %s (from %s) (container=%s / role=%s) (%s)\n", r.URL.Path, r.RemoteAddr, container.Name, role, r.UserAgent())

	if r.Method == "GET" {
		token := r.Header.Get("X-aws-ec2-metadata-token")

		if token != "" {
			timeToken, err := base64.StdEncoding.DecodeString(string(token))

			if err != nil {
				http.Error(w, "", http.StatusUnauthorized)
				return
			}

			t, err := strconv.Atoi(string(timeToken))

			if err != nil {
				http.Error(w, "", http.StatusUnauthorized)
				return
			}

			if int64(t) < time.Now().Unix() {
				http.Error(w, "", http.StatusUnauthorized)
				return
			}
		}
	}

	switch r.URL.Path {
	case "/latest/meta-data/iam/security-credentials":
		fmt.Fprintf(w, "dev")
		return
	case "/latest/meta-data/iam/security-credentials/":
		fmt.Fprintf(w, "dev")
		return
	case "/latest/meta-data/iam/security-credentials/dev":
		sts, err := env.credentialStore.getStsTokenJSON(role)

		if err != nil {
			http.Error(w, fmt.Sprintf("Internal Error: %s", err), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, sts)
		return
	case "/latest/dynamic/instance-identity/document":
		fmt.Fprintf(w, "{\"region\": \"%s\"}", env.region)
		return
	case "/latest/api/token":
		if r.Method == "PUT" {
			ttlStr := r.Header.Get("X-aws-ec2-metadata-token-ttl-seconds")

			if ttlStr == "" {
				fmt.Printf("Request didn't contain X-aws-ec2-metadata-token-ttl-seconds header")
				http.Error(w, "Invalid token", http.StatusBadRequest)
				return
			}

			ttl, err := strconv.Atoi(ttlStr)

			if err != nil {
				fmt.Printf("Parsing X-aws-ec2-metadata-token-ttl-seconds (%s) failed: %s", ttlStr, err)
				http.Error(w, "Invalid token", http.StatusBadRequest)
				return
			}

			timeToken := fmt.Sprintf("%d", time.Now().Unix()+int64(ttl))

			encodedToken := base64.StdEncoding.EncodeToString([]byte(timeToken))
			fmt.Fprintf(w, "%s", encodedToken)
		}
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

func containerFromIp(ip string) (container *SimpleContainer, err error) {
	names, err := net.LookupAddr(ip)

	for _, name := range names {
		shortname := strings.Split(name, ".")[0]
		containerName := fmt.Sprintf("/%s", shortname)

		ctx := context.Background()
		cli, err := client.NewEnvClient()
		if err != nil {
			return nil, err
		}

		_, raw, err := cli.ContainerInspectWithRaw(ctx, containerName, false)

		container = &SimpleContainer{}
		err = json.Unmarshal(raw, container)

		if err != nil {
			log.Printf("ERROR: %s\n", err)
			return nil, err
		}
	}

	return
}

func (container *SimpleContainer) getEnvValue(key string) (value string, ok bool) {
	compareKey := fmt.Sprintf("%s=", key)

	for _, param := range container.Config.Env {
		if strings.HasPrefix(param, compareKey) {
			value := strings.Split(param, "=")[1]

			return value, true
		}
	}

	return "", false
}

func (store *CredentialStore) getValue(arn string) (value *credentials.Value, err error) {
	c, ok := store.credsMap[arn]

	if !ok {
		c = stscreds.NewCredentials(store.sess, arn)
		store.credsMap[arn] = c
	}

	v, err := c.Get()

	return &v, err
}

func (store *CredentialStore) getStsTokenJSON(arn string) (token string, err error) {
	value, err := store.getValue(arn)

	if err != nil {
		return "", err
	}

	sts := ststoken{AccessKeyId: value.AccessKeyID,
		SecretAccessKey: value.SecretAccessKey,
		Token:           value.SessionToken,
		Code:            "Success",
		Type:            "AWS-HMAC",
		Expiration:      time.Now().Add(time.Minute * time.Duration(5)).Format(time.RFC3339),
		LastUpdated:     time.Now().Format(time.RFC3339),
	}

	json, err := json.Marshal(sts)

	if err != nil {
		return "", err
	}

	token = fmt.Sprintf("%s", json)

	return
}
