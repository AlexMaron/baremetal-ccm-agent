package haproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

func (c *Client) bodyJSONLog(ctx context.Context, body any) {
	if c.Log != nil && c.Log.Enabled(ctx, slog.LevelDebug) {
		b, err := json.MarshalIndent(body, "", "  ")
		if err != nil {
			c.Log.Error("failed to marshal backend body", slog.Any("err", err))
		} else {
			fmt.Println(string(b))
		}
	}

}

func (b *BackendRequest) RequiresBackendHTTP() bool {
	if b == nil {
		return false
	}

	switch b.Balance.Algorithm {
	case "hdr", "uri", "uri_param":
		return true
	default:
		return false
	}
}

func (e *ValidationError) Error() string {
	return e.Message
}

func validateBackendByMode(request *BackendRequest) error {
	switch request.Mode {
	case "tcp":
		if request.RequiresBackendHTTP() {
			return &ValidationError{
				Reason:  ReasonUnsupportedCombination,
				Field:   "Balance.Algorithm",
				Message: MsgHTTPBalanceInTCP,
			}
		}
		if request.AdvCheck != "" {
			if request.AdvCheck == "httpchk" {
				return &ValidationError{
					Reason:  ReasonUnsupportedCombination,
					Field:   "AdvCheck",
					Message: MsgHTTPChkInTCP,
				}
			}
		}
	case "http":
		if request.DefaultServer.SendProxy != "" {
			return &ValidationError{
				Reason:  ReasonUnsupportedCombination,
				Field:   "DefaultServer.SendProxy",
				Message: MsgSendProxyInHTTP,
			}
		}
		if request.DefaultServer.SendProxy2 != "" {
			return &ValidationError{
				Reason:  ReasonUnsupportedCombination,
				Field:   "DefaultServer.SendProxy2",
				Message: MsgSendProxy2InHTTP,
			}
		}
		if request.TCPKeepAlive != "" {
			return &ValidationError{
				Reason:  ReasonUnsupportedCombination,
				Field:   "TCPKeepAlive",
				Message: MsgTCPKAInHTTP,
			}
		}
	}

	return nil
}

func validateFronend(request *FrontendRequest) error {
	if request.DefaultBackend == "" {
		return &ValidationError{
			Reason:  ReasonUnsupportedCombination,
			Field:   "Balance.Algorithm",
			Message: MsgDefaultBackendEmpty,
		}
	}
	return nil
}

func checkBackendBodyValues(body BackendRequest) error {
	if body.Name == "" {
		return fmt.Errorf("Backend name is required.")
	}

	if body.Mode == "" {
		return fmt.Errorf("Backend mode is required. Modes: tcp, http.")
	}

	return nil
}


func (b *BackendRequest) UnmarshalJSON(data []byte) error {
    type Alias BackendRequest

    out := &struct{
        *Alias
    }{
        Alias: &Alias{
			Mode:           "tcp",
			Balance:        Balance{
                Algorithm: "leastconn",
            },
			AdvCheck:       "tcp-check",
            DefaultServer: DefaultServer{
                Inter: 3000,
                Fastinter: 1000,
                Fall: 3,
                Rise: 4,
                OnMarkedDown: "shutdown-sessions",
            },
        },
    }

    if err := json.Unmarshal(data, out); err != nil {
        return err
    }

    *b = BackendRequest(*out.Alias)
    return nil
}

func (b *FrontendRequest) UnmarshalJSON(data []byte) error {
    type Alias FrontendRequest

    out := &struct{
        *Alias
    }{
        Alias: &Alias{
            Mode: "tcp",
        },
    }

    if err := json.Unmarshal(data, out); err != nil {
        return err
    }

    *b = FrontendRequest(*out.Alias)
    return nil
}
//func MergeOverride(dst, src any, exclude ...string) error {
//	dstVal := reflect.ValueOf(dst)
//	srcVal := reflect.ValueOf(src)
//
//	if dstVal.Kind() != reflect.Ptr || srcVal.Kind() != reflect.Ptr {
//		return fmt.Errorf("dst and src must be pointers to structs")
//	}
//
//	dstElem := dstVal.Elem()
//	srcElem := srcVal.Elem()
//
//	if dstElem.Kind() != reflect.Struct || srcElem.Kind() != reflect.Struct {
//		return fmt.Errorf("dst and src must point to structs")
//	}
//
//	mergeValue(dstVal.Elem(), srcVal.Elem(), exclude)
//	return nil
//}
//
//func mergeValue(dst, src reflect.Value, exclude []string) {
//	dstType := dst.Type()
//
//	for i := 0; i < dst.NumField(); i++ {
//		fieldName := dstType.Field(i).Name
//
//		if slices.Contains(exclude, fieldName) {
//			continue
//		}
//
//		dstField := dst.Field(i)
//		srcField := src.Field(i)
//
//		if !dstField.CanSet() {
//			continue
//		}
//
//		switch dstField.Kind() {
//		case reflect.Ptr, reflect.Interface:
//			if !srcField.IsNil() {
//				dstField.Set(srcField)
//			}
//		case reflect.Struct:
//			mergeValue(dstField, srcField, exclude)
//		default:
//			zero := reflect.Zero(dstField.Type())
//			if !reflect.DeepEqual(srcField.Interface(), zero.Interface()) {
//				dstField.Set(srcField)
//			}
//		}
//	}
//}
