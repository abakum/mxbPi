package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	maxigobot "github.com/maxigo-bot/maxigo-bot"
	"github.com/maxigo-bot/maxigo-client"
	"golang.org/x/net/proxy"
)

// CreateBotWithProxy создает бота maxigo-bot с поддержкой прокси из переменных окружения
func CreateBotWithProxy(token string) (*maxigobot.Bot, error) {
	// Проверяем наличие токена
	if token == "" {
		return nil, Errorf(dic.add(ul,
			"en:bot token not specified",
			"ru:токен бота не указан",
		))
	}

	// Проверяем переменные окружения для прокси
	proxyEnv := os.Getenv("TMB_PROXY")
	if proxyEnv == "" {
		proxyEnv = os.Getenv("ALL_PROXY")
	}
	if proxyEnv == "" {
		proxyEnv = os.Getenv("SOCKS5_PROXY")
	}

	// Проверяем переменную окружения для API URL
	baseURL := os.Getenv("TMB_URL")

	// Собираем опции для создания клиента
	var opts []maxigobot.Option

	// Если base URL указан, создаем клиента с кастомным URL
	if baseURL != "" {
		fmt.Printf("%s\n", fmt.Sprintf(dic.add(ul,
			"en:Using custom base URL: %s",
			"ru:Использую кастомный base URL: %s",
		), baseURL))

		var client *maxigo.Client
		var err error

		// Если есть прокси, создаем клиента с прокси и с кастомным URL
		if proxyEnv != "" {
			httpClient, err := createHTTPClientWithProxy(proxyEnv)
			if err != nil {
				return nil, err
			}
			client, err = maxigo.New(token, maxigo.WithBaseURL(baseURL), maxigo.WithHTTPClient(httpClient))
		} else {
			client, err = maxigo.New(token, maxigo.WithBaseURL(baseURL))
		}
		if err != nil {
			return nil, err
		}

		opts = append(opts, maxigobot.WithClient(client))
	} else if proxyEnv != "" {
		// Если прокси указан без base URL, создаем клиента с прокси
		fmt.Printf("%s\n", fmt.Sprintf(dic.add(ul,
			"en:Using proxy: %s",
			"ru:Использую прокси: %s",
		), proxyEnv))

		// Создаем HTTP клиент с прокси
		httpClient, err := createHTTPClientWithProxy(proxyEnv)
		if err != nil {
			return nil, Errorf(dic.add(ul,
				"en:failed to create HTTP client with proxy: %w",
				"ru:ошибка создания HTTP клиента с прокси: %w",
			), err)
		}

		// Создаем maxigo-client и передаем HTTP клиент
		client, err := maxigo.New(token, maxigo.WithHTTPClient(httpClient))
		if err != nil {
			return nil, err
		}

		opts = append(opts, maxigobot.WithClient(client))
	} else {
		fmt.Printf("%s\n", dic.add(ul,
			"en:Proxy not specified, creating client without proxy",
			"ru:Прокси не указан, создаю клиента без прокси",
		))
	}

	// Создаем бота maxigo-bot с собранными опциями
	return maxigobot.New(token, opts...)
}

// createHTTPClientWithProxy создает HTTP клиент с поддержкой SOCKS5 прокси
func createHTTPClientWithProxy(proxyStr string) (*http.Client, error) {
	// Удаляем префикс socks5:// если есть
	proxyStr = strings.TrimPrefix(proxyStr, "socks5://")

	// Разделяем хост и порт с использованием стандартной функции
	host, portStr, err := net.SplitHostPort(proxyStr)
	port := "1080" // порт по умолчанию для SOCKS5
	if err != nil {
		// Если порт не указан, используем хост как есть с портом по умолчанию
		// Убираем квадратные скобки для IPv6 адресов
		host = strings.Trim(proxyStr, "[]")
	} else {
		port = portStr // используем указанный порт
	}
	if host == "" {
		return nil, Errorf(dic.add(ul,
			"en:host not specified",
			"ru:хост не указан",
		))
	}

	// Проверяем порт
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return nil, Errorf(dic.add(ul,
			"en:invalid port: %s",
			"ru:неверный порт: %s",
		), port)
	}

	// Создаем SOCKS5 dialer
	dialer, err := proxy.SOCKS5("tcp", net.JoinHostPort(host, port), nil, proxy.Direct)
	if err != nil {
		return nil, Errorf(dic.add(ul,
			"en:failed to create SOCKS5 dialer: %w",
			"ru:ошибка создания SOCKS5 dialer: %w",
		), err)
	}

	// Создаем транспорт с SOCKS5 dialer
	transport := &http.Transport{
		Dial: dialer.Dial,
		// Таймауты для надежности
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
		MaxIdleConns:       100,
		IdleConnTimeout:    90 * time.Second,
		DisableCompression: true,
	}

	// Создаем HTTP клиент
	client := &http.Client{
		Transport: transport,
		Timeout:   refresh,
	}

	return client, nil
}
