package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"memfab/internal/config"
)

const maxConfigCommandBodyBytes = 1 << 20

func configCommand(args []string, input io.Reader, output io.Writer) error {
	if len(args) == 0 {
		if _, err := config.WriteDefaultIfMissing(); err != nil {
			return err
		}
		text, err := config.RedactedTOML()
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(output, text)
		return err
	}

	switch args[0] {
	case "get":
		if len(args) != 2 || args[1] != "--json" {
			return errors.New("usage: memfab config get --json")
		}
		cfg, err := config.ReadUI()
		if err != nil {
			return err
		}
		return encodeConfigJSON(output, cfg)
	case "update":
		if len(args) != 2 || args[1] != "--json" {
			return errors.New("usage: memfab config update --json")
		}
		var request config.UIConfig
		if err := decodeConfigJSON(input, &request); err != nil {
			return err
		}
		transaction, err := config.UpdateUITransaction(request)
		if err != nil {
			return err
		}
		if err := encodeConfigJSON(output, transaction.UIUpdateResult); err != nil {
			if rollbackErr := transaction.Rollback(); rollbackErr != nil {
				return fmt.Errorf("encode config update result: %w; rollback failed: %v", err, rollbackErr)
			}
			return fmt.Errorf("encode config update result: %w", err)
		}
		return nil
	case "revision":
		if len(args) != 2 || args[1] != "--raw" {
			return errors.New("usage: memfab config revision --raw")
		}
		data, err := readConfigCommandRaw(input, "revision")
		if err != nil {
			return err
		}
		revision, err := config.Revision(data)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(output, revision)
		return err
	case "restore":
		if len(args) != 4 || args[1] != "--raw" || args[2] != "--if-revision" || args[3] == "" {
			return errors.New("usage: memfab config restore --raw --if-revision REVISION")
		}
		data, err := readConfigCommandRaw(input, "restore")
		if err != nil {
			return err
		}
		return config.Restore(data, args[3])
	default:
		return fmt.Errorf("unknown config command %s", args[0])
	}
}

func readConfigCommandRaw(input io.Reader, operation string) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(input, maxConfigCommandBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("config %s payload is empty", operation)
	}
	if len(data) > maxConfigCommandBodyBytes {
		return nil, fmt.Errorf("config %s payload is too large", operation)
	}
	return data, nil
}

func decodeConfigJSON(input io.Reader, value interface{}) error {
	limited := io.LimitReader(input, maxConfigCommandBodyBytes+1)
	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("decode config JSON: %w", err)
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("decode config JSON: multiple values are not allowed")
		}
		return fmt.Errorf("decode config JSON: %w", err)
	}
	return nil
}

func encodeConfigJSON(output io.Writer, value interface{}) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
