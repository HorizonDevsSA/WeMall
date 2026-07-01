package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/wemall/seller-service/internal/crypto"
	"github.com/wemall/seller-service/internal/db"
	"github.com/wemall/seller-service/internal/service"
)

// Mock DBTX
type mockRow struct {
	scanFunc func(dest ...any) error
}

func (r mockRow) Scan(dest ...any) error {
	if r.scanFunc != nil {
		return r.scanFunc(dest...)
	}
	return nil
}

type mockDBTX struct {
	queryRowFunc func(ctx context.Context, query string, args ...any) pgx.Row
}

func (m mockDBTX) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(""), nil
}

func (m mockDBTX) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func (m mockDBTX) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, query, args...)
	}
	return mockRow{}
}

func TestSellerService_SecurityPINAndEncryption(t *testing.T) {
	ctx := context.Background()
	key := "12345678901234567890123456789012"
	enc, err := crypto.NewEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	userID := uuid.New()
	sellerID := uuid.New()
	correctPIN := "123456"

	// Mock DBTX behavior
	var storedPinHash string
	var storedBankName, storedBankAccount, storedEcocash string

	dbtx := mockDBTX{
		queryRowFunc: func(ctx context.Context, query string, args ...any) pgx.Row {
			if strings.Contains(query, "GetSellerByUserID") {
				return mockRow{
					scanFunc: func(dest ...any) error {
						if len(dest) > 34 {
							*dest[0].(*uuid.UUID) = sellerID
							*dest[1].(*uuid.UUID) = userID
							*dest[2].(*string) = "My Store"
							*dest[19].(*string) = storedBankName
							*dest[20].(*string) = storedBankAccount
							*dest[21].(*string) = storedEcocash
							if storedPinHash != "" {
								p := storedPinHash
								*dest[34].(**string) = &p
							} else {
								*dest[34].(**string) = nil
							}
						}
						return nil
					},
				}
			}
			if strings.Contains(query, "UpdateSellerPIN") {
				pinHashText := args[1].(*string)
				if pinHashText != nil {
					storedPinHash = *pinHashText
				}
				return mockRow{
					scanFunc: func(dest ...any) error {
						if len(dest) > 34 {
							*dest[0].(*uuid.UUID) = sellerID
							*dest[1].(*uuid.UUID) = userID
							if storedPinHash != "" {
								p := storedPinHash
								*dest[34].(**string) = &p
							} else {
								*dest[34].(**string) = nil
							}
						}
						return nil
					},
				}
			}
			if strings.Contains(query, "UpdateSeller") && !strings.Contains(query, "PIN") && !strings.Contains(query, "SecurityCooldown") {
				bankNameArg := args[9].(*string)
				bankAccountArg := args[10].(*string)
				ecocashArg := args[11].(*string)

				if bankNameArg != nil {
					storedBankName = *bankNameArg
				}
				if bankAccountArg != nil {
					storedBankAccount = *bankAccountArg
				}
				if ecocashArg != nil {
					storedEcocash = *ecocashArg
				}

				return mockRow{
					scanFunc: func(dest ...any) error {
						if len(dest) > 34 {
							*dest[0].(*uuid.UUID) = sellerID
							*dest[1].(*uuid.UUID) = userID
							*dest[19].(*string) = storedBankName
							*dest[20].(*string) = storedBankAccount
							*dest[21].(*string) = storedEcocash
							if storedPinHash != "" {
								p := storedPinHash
								*dest[34].(**string) = &p
							} else {
								*dest[34].(**string) = nil
							}
						}
						return nil
					},
				}
			}
			if strings.Contains(query, "UpdateSellerSecurityCooldown") {
				return mockRow{
					scanFunc: func(dest ...any) error {
						if len(dest) > 34 {
							*dest[0].(*uuid.UUID) = sellerID
							*dest[1].(*uuid.UUID) = userID
							*dest[19].(*string) = storedBankName
							*dest[20].(*string) = storedBankAccount
							*dest[21].(*string) = storedEcocash
							if storedPinHash != "" {
								p := storedPinHash
								*dest[34].(**string) = &p
							}
						}
						return nil
					},
				}
			}
			return mockRow{}
		},
	}

	queries := db.New(dbtx)
	svc := service.NewSellerService(queries, nil, enc)

	// --- 1. Test update bank details without a PIN set ---
	_, err = svc.UpdateStore(ctx, service.UpdateStoreInput{
		UserID:            userID,
		BankName:          strPtr("Steward Bank"),
		BankAccountNumber: strPtr("1234567890"),
	})
	if err == nil || !strings.Contains(err.Error(), "a security PIN must be set") {
		t.Errorf("expected error 'a security PIN must be set', got: %v", err)
	}

	// --- 2. Set the Seller PIN ---
	err = svc.SetSellerPIN(ctx, userID, correctPIN)
	if err != nil {
		t.Fatalf("failed to set PIN: %v", err)
	}
	if storedPinHash == "" {
		t.Fatal("expected PIN hash to be saved in DB")
	}

	// --- 3. Attempt update with no PIN supplied ---
	_, err = svc.UpdateStore(ctx, service.UpdateStoreInput{
		UserID:            userID,
		BankName:          strPtr("Steward Bank"),
		BankAccountNumber: strPtr("1234567890"),
	})
	if err == nil || !strings.Contains(err.Error(), "security PIN is required") {
		t.Errorf("expected error 'security PIN is required', got: %v", err)
	}

	// --- 4. Attempt update with incorrect PIN ---
	wrongPIN := "654321"
	_, err = svc.UpdateStore(ctx, service.UpdateStoreInput{
		UserID:            userID,
		BankName:          strPtr("Steward Bank"),
		BankAccountNumber: strPtr("1234567890"),
		PIN:               &wrongPIN,
	})
	if err == nil || !strings.Contains(err.Error(), "invalid security PIN") {
		t.Errorf("expected error 'invalid security PIN', got: %v", err)
	}

	// --- 5. Update with correct PIN (Success) ---
	updatedSeller, err := svc.UpdateStore(ctx, service.UpdateStoreInput{
		UserID:            userID,
		BankName:          strPtr("Steward Bank"),
		BankAccountNumber: strPtr("1234567890"),
		EcocashNumber:     strPtr("077111222"),
		PIN:               &correctPIN,
	})
	if err != nil {
		t.Fatalf("unexpected error updating bank details: %v", err)
	}

	// Verify that service returned the plaintext values (transparency inside service layer)
	if updatedSeller.BankName != "Steward Bank" || updatedSeller.BankAccountNumber != "1234567890" || updatedSeller.EcocashNumber != "077111222" {
		t.Errorf("expected plaintext bank details in service return, got Name=%q, Acc=%q, Eco=%q", updatedSeller.BankName, updatedSeller.BankAccountNumber, updatedSeller.EcocashNumber)
	}

	// Verify that database values are encrypted
	if !strings.HasPrefix(storedBankName, "aes256gcm:") || !strings.HasPrefix(storedBankAccount, "aes256gcm:") || !strings.HasPrefix(storedEcocash, "aes256gcm:") {
		t.Errorf("expected database stored values to be encrypted, got Name=%q, Acc=%q, Eco=%q", storedBankName, storedBankAccount, storedEcocash)
	}

	// --- 6. Reveal Bank Details (Incorrect PIN) ---
	_, _, _, err = svc.RevealBankDetails(ctx, userID, wrongPIN)
	if err == nil || !strings.Contains(err.Error(), "invalid security PIN") {
		t.Errorf("expected reveal error 'invalid security PIN', got: %v", err)
	}

	// --- 7. Reveal Bank Details (Correct PIN) ---
	decName, decAcc, decEco, err := svc.RevealBankDetails(ctx, userID, correctPIN)
	if err != nil {
		t.Fatalf("failed to reveal bank details: %v", err)
	}
	if decName != "Steward Bank" || decAcc != "1234567890" || decEco != "077111222" {
		t.Errorf("expected revealed plaintext bank details, got Name=%q, Acc=%q, Eco=%q", decName, decAcc, decEco)
	}
}

func strPtr(s string) *string {
	return &s
}
