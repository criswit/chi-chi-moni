package db

import (
	"fmt"

	"github.com/criswit/chi-chi-moni/model"
	"github.com/jmoiron/sqlx"
)

const bankAccountTable = "BANK_ACCOUNT"
const bankAccountBalanceTable = "BANK_ACCOUNT_BALANCE"

type DatabaseClient struct {
	db *sqlx.DB
}

func NewDatabaseClient(path string) (*DatabaseClient, error) {
	db, err := sqlx.Connect("sqlite3", path)
	if err != nil {
		return nil, err
	}
	return &DatabaseClient{db: db}, nil
}

func (c *DatabaseClient) Close() {
	c.db.Close()
}

func (c *DatabaseClient) PutBankAccount(account model.Account) error {
	query := fmt.Sprintf("INSERT INTO %s (ID, NAME, INSTITUTION_NAME) VALUES (?, ?, ?)", bankAccountTable)
	_, err := c.db.Exec(query, account.ID, account.Name, account.Org.Name)
	if err != nil {
		return err
	}
	return nil
}

func (c *DatabaseClient) PutBankAccountBalance(bankAccountId string, runId string, balance string) error {
	query := fmt.Sprintf("INSERT INTO %s (ID, RUN_ID, BALANCE) VALUES (?, ?, ?)", bankAccountBalanceTable)
	_, err := c.db.Exec(query, bankAccountId, runId, balance)
	if err != nil {
		return err
	}
	return nil
}

func (c *DatabaseClient) PutAccountBalance(bankAccountId string, runId string, balance string) error {
	query := "INSERT INTO BANK_ACCOUNT_BALANCE (BANK_ACCOUNT_ID, RUN_ID, BALANCE) VALUES (?, ?, ?)"
	_, err := c.db.Exec(query, bankAccountId, runId, balance)
	if err != nil {
		return err
	}
	return nil
}

func (c *DatabaseClient) DoesBankAccountExist(accountId string) (bool, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE ID = ?", bankAccountTable)
	var count int
	err := c.db.Get(&count, query, accountId)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
