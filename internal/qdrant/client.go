package qdrant

import (
	"context"
	"fmt"
	"math"
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

type CollectionInfo struct {
	VectorCount  int64
	IndexedCount int64
	SizeBytes    int64
}

type UpsertPoint struct {
	ID      string
	Vector  []float32
	Payload map[string]any
}

type SearchResult struct {
	ID      string
	Score   float32
	Payload map[string]any
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

	response, err := c.client.GetCollectionsClient().List(ctx, &qdrant.ListCollectionsRequest{})
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

func (c *Client) CreateCollection(ctx context.Context, name string, dimension int, metric string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("collection name is required")
	}
	if dimension <= 0 {
		return fmt.Errorf("dimension must be positive")
	}

	err := c.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: name,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(dimension),
			Distance: parseDistance(metric),
		}),
	})
	if err != nil {
		return fmt.Errorf("create qdrant collection failed: %w", err)
	}

	return nil
}

func (c *Client) DeleteCollection(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("collection name is required")
	}
	if err := c.client.DeleteCollection(ctx, name); err != nil {
		return fmt.Errorf("delete qdrant collection failed: %w", err)
	}
	return nil
}

func (c *Client) GetCollectionInfo(ctx context.Context, name string) (*CollectionInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("collection name is required")
	}

	info, err := c.client.GetCollectionInfo(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get qdrant collection info failed: %w", err)
	}
	pointsCount := info.GetPointsCount()
	if pointsCount > math.MaxInt64 {
		pointsCount = math.MaxInt64
	}
	indexedCount := info.GetIndexedVectorsCount()
	if indexedCount > math.MaxInt64 {
		indexedCount = math.MaxInt64
	}
	vectorCountSafe := parseUint64ToInt64(pointsCount)
	indexedCountSafe := parseUint64ToInt64(indexedCount)

	return &CollectionInfo{
		VectorCount:  vectorCountSafe,
		IndexedCount: indexedCountSafe,
		SizeBytes:    0,
	}, nil
}

func parseUint64ToInt64(value uint64) int64 {
	parsed, err := strconv.ParseInt(strconv.FormatUint(value, 10), 10, 64)
	if err != nil {
		return math.MaxInt64
	}
	return parsed
}

func (c *Client) UpsertPoints(ctx context.Context, collectionName string, points []UpsertPoint) error {
	collection := strings.TrimSpace(collectionName)
	if collection == "" {
		return fmt.Errorf("collection name is required")
	}
	if len(points) == 0 {
		return fmt.Errorf("points are required")
	}

	qdrantPoints := make([]*qdrant.PointStruct, 0, len(points))
	for idx := range points {
		id := strings.TrimSpace(points[idx].ID)
		if id == "" {
			return fmt.Errorf("point id is required")
		}
		if len(points[idx].Vector) == 0 {
			return fmt.Errorf("point vector is required")
		}
		qdrantPoints = append(qdrantPoints, &qdrant.PointStruct{
			Id:      qdrant.NewID(id),
			Vectors: qdrant.NewVectors(points[idx].Vector...),
			Payload: qdrant.NewValueMap(points[idx].Payload),
		})
	}

	wait := true
	if _, err := c.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collection,
		Wait:           &wait,
		Points:         qdrantPoints,
	}); err != nil {
		return fmt.Errorf("upsert qdrant points failed: %w", err)
	}

	return nil
}

func (c *Client) Search(ctx context.Context, collectionName string, vector []float32, topK int, minScore float32) ([]SearchResult, error) {
	collection := strings.TrimSpace(collectionName)
	if collection == "" {
		return nil, fmt.Errorf("collection name is required")
	}
	if len(vector) == 0 {
		return nil, fmt.Errorf("vector is required")
	}
	if topK <= 0 {
		return nil, fmt.Errorf("top_k must be positive")
	}

	limit := uint64(topK)
	query := &qdrant.QueryPoints{
		CollectionName: collection,
		Query:          qdrant.NewQuery(vector...),
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true),
	}
	if minScore > 0 {
		threshold := minScore
		query.ScoreThreshold = &threshold
	}

	scored, err := c.client.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query qdrant points failed: %w", err)
	}

	results := make([]SearchResult, 0, len(scored))
	for _, point := range scored {
		if point == nil {
			continue
		}
		results = append(results, SearchResult{
			ID:      pointIDToString(point.GetId()),
			Score:   point.GetScore(),
			Payload: payloadToAnyMap(point.GetPayload()),
		})
	}

	return results, nil
}

func (c *Client) GetByID(ctx context.Context, collectionName, id string) (*SearchResult, error) {
	collection := strings.TrimSpace(collectionName)
	if collection == "" {
		return nil, fmt.Errorf("collection name is required")
	}
	key := strings.TrimSpace(id)
	if key == "" {
		return nil, fmt.Errorf("id is required")
	}

	retrieved, err := c.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: collection,
		Ids:            []*qdrant.PointId{qdrant.NewID(key)},
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("get qdrant point failed: %w", err)
	}
	if len(retrieved) == 0 {
		return nil, fmt.Errorf("point not found")
	}

	point := retrieved[0]
	return &SearchResult{
		ID:      pointIDToString(point.GetId()),
		Score:   0,
		Payload: payloadToAnyMap(point.GetPayload()),
	}, nil
}

func pointIDToString(id *qdrant.PointId) string {
	if id == nil {
		return ""
	}
	if uuid := strings.TrimSpace(id.GetUuid()); uuid != "" {
		return uuid
	}
	return strconv.FormatUint(id.GetNum(), 10)
}

func payloadToAnyMap(payload map[string]*qdrant.Value) map[string]any {
	if len(payload) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(payload))
	for key, value := range payload {
		result[key] = qdrantValueToAny(value)
	}
	return result
}

func qdrantValueToAny(value *qdrant.Value) any {
	if value == nil {
		return nil
	}
	switch kind := value.GetKind().(type) {
	case *qdrant.Value_NullValue:
		return nil
	case *qdrant.Value_BoolValue:
		return kind.BoolValue
	case *qdrant.Value_IntegerValue:
		return kind.IntegerValue
	case *qdrant.Value_DoubleValue:
		return kind.DoubleValue
	case *qdrant.Value_StringValue:
		return kind.StringValue
	case *qdrant.Value_StructValue:
		fields := kind.StructValue.GetFields()
		mapped := make(map[string]any, len(fields))
		for key, fieldValue := range fields {
			mapped[key] = qdrantValueToAny(fieldValue)
		}
		return mapped
	case *qdrant.Value_ListValue:
		values := kind.ListValue.GetValues()
		mapped := make([]any, 0, len(values))
		for _, item := range values {
			mapped = append(mapped, qdrantValueToAny(item))
		}
		return mapped
	default:
		return nil
	}
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

func parseDistance(metric string) qdrant.Distance {
	switch strings.ToLower(strings.TrimSpace(metric)) {
	case "dot":
		return qdrant.Distance_Dot
	case "euclid", "euclidean", "l2":
		return qdrant.Distance_Euclid
	case "manhattan", "l1":
		return qdrant.Distance_Manhattan
	default:
		return qdrant.Distance_Cosine
	}
}
