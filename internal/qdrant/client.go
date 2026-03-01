package qdrant

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"ai-gateway/pkg/logger"

	"github.com/sirupsen/logrus"

	"github.com/qdrant/go-client/qdrant"
)

var qdrantLogger = logger.WithField("component", "qdrant")

type Client struct {
	client         *qdrant.Client
	httpAddr       string
	apiKey         string
	collectionName string
}

func NewQdrantClient(httpAddr, apiKey, collectionName string) (*Client, error) {
	host, port, useTLS, err := parseHTTPAddr(httpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTTP address: %w", err)
	}

	grpcPort := port
	if grpcPort == 0 {
		grpcPort = 6334
	}

	qdrantLogger.WithFields(logrus.Fields{
		"host":       host,
		"port":       grpcPort,
		"use_tls":    useTLS,
		"api_key":    maskAPIKey(apiKey),
		"collection": collectionName,
	}).Info("Creating Qdrant client")

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   host,
		Port:   grpcPort,
		APIKey: apiKey,
		UseTLS: useTLS,
	})
	if err != nil {
		qdrantLogger.WithError(err).Error("Failed to create Qdrant client")
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	qdrantLogger.Info("Qdrant client created successfully")

	return &Client{
		client:         client,
		httpAddr:       httpAddr,
		apiKey:         apiKey,
		collectionName: collectionName,
	}, nil
}

func (c *Client) Close() error {
	qdrantLogger.Info("Closing Qdrant client connection")
	if err := c.client.Close(); err != nil {
		qdrantLogger.WithError(err).Error("Failed to close Qdrant client")
		return fmt.Errorf("failed to close Qdrant client: %w", err)
	}
	qdrantLogger.Info("Qdrant client closed successfully")
	return nil
}

func (c *Client) Health(ctx context.Context) error {
	qdrantLogger.Debug("Checking Qdrant connection health")

	collectionsClient := c.client.GetCollectionsClient()

	_, err := collectionsClient.List(ctx, &qdrant.ListCollectionsRequest{})
	if err != nil {
		qdrantLogger.WithError(err).Error("Qdrant health check failed")
		return fmt.Errorf("Qdrant health check failed: %w", err)
	}

	qdrantLogger.Debug("Qdrant health check passed")
	return nil
}

func (c *Client) GetCollections(ctx context.Context) ([]string, error) {
	qdrantLogger.Debug("Listing Qdrant collections")

	collectionsClient := c.client.GetCollectionsClient()

	response, err := collectionsClient.List(ctx, &qdrant.ListCollectionsRequest{})
	if err != nil {
		qdrantLogger.WithError(err).Error("Failed to list collections")
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	collections := make([]string, 0, len(response.GetCollections()))
	for _, col := range response.GetCollections() {
		collections = append(collections, col.GetName())
	}

	qdrantLogger.WithField("count", len(collections)).Debug("Retrieved collections list")
	return collections, nil
}

func parseHTTPAddr(httpAddr string) (host string, port int, useTLS bool, err error) {
	if httpAddr == "" {
		return "localhost", 6334, false, nil
	}

	addr := httpAddr
	if strings.HasPrefix(addr, "http://") {
		addr = strings.TrimPrefix(addr, "http://")
		useTLS = false
	} else if strings.HasPrefix(addr, "https://") {
		addr = strings.TrimPrefix(addr, "https://")
		useTLS = true
	}

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, 0, useTLS, nil
	}

	if portStr != "" {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return "", 0, false, fmt.Errorf("invalid port: %s", portStr)
		}
	}

	return host, port, useTLS, nil
}

func maskAPIKey(apiKey string) string {
	if apiKey == "" {
		return "none"
	}
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "***" + apiKey[len(apiKey)-4:]
}
