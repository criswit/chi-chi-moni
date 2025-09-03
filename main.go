package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/criswit/chi-chi-moni/api"
	"github.com/criswit/chi-chi-moni/aws"
	"github.com/criswit/chi-chi-moni/db"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

const ssoProfile = "monkstorage"
const accessTokenSecretName = "monk-monies"
const dbFilePath = "data/monk.db"

func getAccessToken() (accessToken api.AccessToken, err error) {
	ssoClient, err := aws.NewSSOClient(ssoProfile, "us-east-1")
	if err != nil {
		return api.AccessToken{}, err
	}
	secretClient, err := aws.NewSecretsManagerClientWithSSO(context.Background(), ssoClient)
	if err != nil {
		return api.AccessToken{}, err
	}
	return secretClient.RetrieveAccessToken(context.Background(), accessTokenSecretName)
}

func getDbFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, dbFilePath), nil
}

func main() {
	accessToken, err := getAccessToken()
	if err != nil {
		log.Fatal(err)
	}
	finClient, err := api.NewSimpleFinClient(accessToken)
	if err != nil {
		log.Fatal(err)
	}

	dbPath, err := getDbFilePath()
	if err != nil {
		log.Fatal(err)
	}

	dbClient, err := db.NewDatabaseClient(dbPath)
	if err != nil {
		log.Fatal(err)
	}

	jobUuid := uuid.New()

	getAccountsResp, err := finClient.GetAccounts(&api.GetAccountsOptions{})
	if err != nil {
		log.Fatal(err)
	}

	for _, account := range getAccountsResp.Accounts {
		exists, err := dbClient.DoesBankAccountExist(account.ID)
		if err != nil {
			log.Fatal(err)
		}
		if !exists {
			if err = dbClient.PutBankAccount(account); err != nil {
				log.Fatal(err)
			}
		}

		if err := dbClient.PutAccountBalance(account.ID, jobUuid.String(), account.Balance); err != nil {
			log.Fatal(err)
		}
	}
}
