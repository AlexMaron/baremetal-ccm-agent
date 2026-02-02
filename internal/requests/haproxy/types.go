package haproxy

import (
	"log/slog"
	"net/http"
)

const (
	ServerCheckEnabled  Check = "enabled"
	ServerCheckDisabled Check = "disabled"
)

const (
	ReasonUnsupportedCombination = "UnsupportedCombination"

	MsgHTTPBalanceInTCP = "http-based balance algorithm is not allowed in tcp mode"
	MsgHTTPChkInTCP     = "httpchk is not allowed in tcp mode"

	MsgSendProxyInHTTP  = "send-proxy is not allowed in http mode"
	MsgSendProxy2InHTTP = "send-proxy-v2 is not allowed in http mode"
	MsgTCPKAInHTTP      = "tcpka is not allowed in http mode"

	// Frontend Messages
	MsgDefaultBackendEmpty = "default-backend empty"
)

type Check string

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Username   string
	Password   string
	Log        *slog.Logger
}

type ValidationError struct {
	Reason  string
	Message string
	Field   string
}

type ValidationErrors struct {
	Errors []*ValidationError
}

type Balance struct {
	// Allowed: first┃hash┃hdr┃leastconn┃random┃rdp-cookie┃roundrobin┃source┃static-rr┃uri┃url_param
	Algorithm string `json:"algorithm"`
}

type DefaultServer struct {
	// https://www.haproxy.com/documentation/haproxy-configuration-manual/latest/#5.2-inter
	// inter <delay> — интервал между обычными health check в миллисекундах (по умолчанию 2000 ms).
	// Используется, когда сервер в стабильном состоянии UP.
	Inter int64 `json:"inter"`

	// fastinter <delay> — сокращённый интервал для промежуточных состояний сервера
	// (например, когда он транзитом уходит в DOWN или восстанавливается).
	Fastinter int64 `json:"fastinter"`

	// fall <count> — сервер считается неработоспособным (DOWN) после <count>
	// подряд неудачных health check (по умолчанию 3).
	// Используется вместе с check, inter и rise для настройки проверки состояния серверов.
	Fall int64 `json:"fall"`
	Rise int64 `json:"rise"`

	// Currently one action is available: shutdown-sessions
	OnMarkedDown string `json:"on-marked-down"`

	// https://www.haproxy.com/documentation/haproxy-configuration-manual/latest/#send-proxy
	// send-proxy заставляет HAProxy использовать PROXY protocol при соединении с сервером,
	// передавая реальный IP клиента и адрес, к которому подключились.
	// Сервер должен поддерживать PROXY protocol, иначе это сломает соединение.
	// Подходит для цепочек HAProxy и TCPv4/6 соединений. Для health check PROXY protocol
	// включается автоматически, если не указаны port или addr.
	SendProxy string `json:"send-proxy,omitempty"`

	// Тоже самое что и send-proxy но с поддержкой расширенной информации и ALPN.
	SendProxy2 string `json:"send-proxy-v2,omitempty"`
}

type BackendRequest struct {
	Name string `json:"name"` // Имя бэкенда
	Mode string `json:"mode"` // Режим работы (tcp, http)

	// Определяет алгоритм балансировки нагрузки для бэкенда.
	// Может использоваться в контекстах: tcp, http, log
	Balance Balance `json:"balance"`

	// Дополнительные проверки сервера.
	// Допустимые значения: httpchk┃ldap-check┃mysql-check┃pgsql-check┃redis-check┃smtpchk┃ssl-hello-chk┃tcp-check
	AdvCheck string `json:"adv_check"`

	// Изменение опций по умолчанию для серверов в бэкенде
	// Может использоваться в контекстах: tcp, http
	DefaultServer DefaultServer `json:"default_server"`

	// Включение или отключение отправки TCP keepalive пакетов со стороны клиента
	// Может использоваться в контекстах: tcp, http
	TCPKeepAlive string `json:"tcpka,omitempty"`

	// Включение вставки заголовка X-Forwarded-For в запросы к серверам
	// Может использоваться в контекстах: http
	ForwardFor *ForwardFor `json:"forwardfor,omitempty"`

	// Установка максимального времени ожидания для успешного подключения к серверу
	// Может использоваться в контекстах: tcp, http, log
	ConnectTimeout string `json:"connect_timeout,omitempty"`

	// Установка максимального времени бездействия на стороне сервера
	// Может использоваться в контекстах: tcp, http, log
	ServerTimeout string `json:"server_timeout,omitempty"`
}

type ForwardFor struct {
	Enabled string `json:"enabled"` // Включение/отключение вставки X-Forwarded-For

	// Можно исключить добавление заголовка для определённого IP или сети,
	// указав "except" и адрес сети. В этом случае запросы с этими IP не будут получать заголовок.
	// Чаще всего используется для приватных сетей или 127.0.0.1. Поддерживаются IPv4 и IPv6.
	Except string `json:"except,omitempty"`

	// Имя конкретного заголовка, который нужно вставлять.
	Header string `json:"header,omitempty"`

	// Ключевое слово "if-none" означает, что заголовок будет добавлен
	// только если его ещё нет. Использовать только в полностью доверенной среде,
	// иначе это может создать уязвимость, если заголовки, поступающие в HAProxy,
	// контролируются пользователем.
	IfNone bool `json:"ifnone,omitempty"`
}

type ServerRequest struct {
	Name    string `json:"Name"`
	Address string `json:"address"`
	Port    int32  `json:"port"`
	Check   Check  `json:"check"`
}

type FrontendRequest struct {
	Name           string `json:"name"`
	DefaultBackend string `json:"default_backend"`
	Mode           string `json:"mode,omitempty"`

	// Устанавливает максимальное время бездействия на стороне клиента.
	// Может использоваться в следующих контекстах: tcp, http
	ClientTimeout string `json:"client_timeout,omitempty"`
}

type FrontendBindRequest struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int32  `json:"port"`
}
