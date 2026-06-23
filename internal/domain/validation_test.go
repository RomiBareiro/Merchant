package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateTransactionCommand(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		command CreateTransactionCommand
		wantErr error
	}{
		{
			name: "Given an empty description when validating a transaction command then returns invalid transaction description",
			command: CreateTransactionCommand{
				Value:              "100",
				Description:        "",
				Method:             string(PaymentMethodDebitCard),
				CardNumber:         "1234 5678 9012 3456",
				CardHolderName:     "John Doe",
				CardExpirationDate: "04/28",
				CardCvv:            "123",
			},
			wantErr: ErrInvalidTransactionDescription,
		},
		{
			name: "Given an invalid amount when validating a transaction command then returns invalid transaction value",
			command: CreateTransactionCommand{
				Value:              "abc",
				Description:        "Valid",
				Method:             string(PaymentMethodDebitCard),
				CardNumber:         "1234 5678 9012 3456",
				CardHolderName:     "John Doe",
				CardExpirationDate: "04/28",
				CardCvv:            "123",
			},
			wantErr: ErrInvalidTransactionValue,
		},
		{
			name: "Given an invalid payment method when validating a transaction command then returns invalid payment method",
			command: CreateTransactionCommand{
				Value:              "100",
				Description:        "Valid",
				Method:             "boleto",
				CardNumber:         "1234 5678 9012 3456",
				CardHolderName:     "John Doe",
				CardExpirationDate: "04/28",
				CardCvv:            "123",
			},
			wantErr: ErrInvalidPaymentMethod,
		},
		{
			name: "Given an invalid card number when validating a transaction command then returns invalid card number",
			command: CreateTransactionCommand{
				Value:              "100",
				Description:        "Valid",
				Method:             string(PaymentMethodDebitCard),
				CardNumber:         "12",
				CardHolderName:     "John Doe",
				CardExpirationDate: "04/28",
				CardCvv:            "123",
			},
			wantErr: ErrInvalidCardNumber,
		},
		{
			name: "Given an invalid expiration date when validating a transaction command then returns invalid card expiration date",
			command: CreateTransactionCommand{
				Value:              "100",
				Description:        "Valid",
				Method:             string(PaymentMethodDebitCard),
				CardNumber:         "1234 5678 9012 3456",
				CardHolderName:     "John Doe",
				CardExpirationDate: "0428",
				CardCvv:            "123",
			},
			wantErr: ErrInvalidCardExpirationDate,
		},
		{
			name: "Given an invalid CVV when validating a transaction command then returns invalid card cvv",
			command: CreateTransactionCommand{
				Value:              "100",
				Description:        "Valid",
				Method:             string(PaymentMethodDebitCard),
				CardNumber:         "1234 5678 9012 3456",
				CardHolderName:     "John Doe",
				CardExpirationDate: "04/28",
				CardCvv:            "12",
			},
			wantErr: ErrInvalidCardCvv,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.command.Validate()

			switch tc.wantErr {
			case nil:
				require.NoError(t, err)
			default:
				require.Equal(t, tc.wantErr, err)
			}
		})
	}
}

func TestParseAmount(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{
			name:    "Given a valid decimal amount when parsing then returns formatted amount",
			value:   "100.00",
			want:    "100.00",
			wantErr: false,
		},
		{
			name:    "Given a valid integer amount when parsing then returns formatted amount",
			value:   " 50 ",
			want:    "50.00",
			wantErr: false,
		},
		{
			name:    "Given an empty amount when parsing then returns an error",
			value:   "",
			wantErr: true,
		},
		{
			name:    "Given an invalid amount string when parsing then returns an error",
			value:   "abc",
			wantErr: true,
		},
		{
			name:    "Given a negative amount when parsing then returns an error",
			value:   "-10",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := ParseAmount(tc.value)

			switch tc.wantErr {
			case true:
				require.Error(t, err)
			default:
				require.NoError(t, err)
				require.Equal(t, tc.want, FormatAmount(result))
			}
		})
	}
}

func TestLast4CardDigits(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		card string
		want string
	}{
		{
			name: "Given a card number with spaces when extracting last digits then returns last 4 digits",
			card: "1234 5678 9012 3456",
			want: "3456",
		},
		{
			name: "Given a short card number when extracting last digits then returns the full number",
			card: "123",
			want: "123",
		},
		{
			name: "Given a card number with punctuation when extracting last digits then ignores non-digits",
			card: "abcd-1234",
			want: "1234",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			switch tc.name {
			case "Given a card number with spaces when extracting last digits then returns last 4 digits",
				"Given a short card number when extracting last digits then returns the full number",
				"Given a card number with punctuation when extracting last digits then ignores non-digits":
				require.Equal(t, tc.want, Last4CardDigits(tc.card))
			default:
				t.Fatalf("unexpected test case %q", tc.name)
			}
		})
	}
}
