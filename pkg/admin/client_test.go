package admin

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	szk "github.com/samuel/go-zookeeper/zk"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/topicctl/pkg/util"
	"github.com/segmentio/topicctl/pkg/zk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetClusterID(t *testing.T) {
	zkConn, _, err := szk.Connect(
		[]string{util.TestZKAddr()},
		5*time.Second,
	)
	require.Nil(t, err)
	require.NotNil(t, zkConn)
	defer zkConn.Close()

	clusterName := testClusterID("clusterID")
	zk.CreateNodes(
		t,
		zkConn,
		[]zk.PathTuple{
			{
				Path: fmt.Sprintf("/%s", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/cluster", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/cluster/id", clusterName),
				Obj: map[string]interface{}{
					"version": "1",
					"id":      "test-cluster-id",
				},
			},
		},
	)

	ctx := context.Background()
	adminClient, err := NewClient(
		ctx,
		ClientConfig{
			ZKAddrs:           []string{util.TestZKAddr()},
			ZKPrefix:          clusterName,
			BootstrapAddrs:    []string{util.TestKafkaAddr()},
			ExpectedClusterID: "test-cluster-id",
			ReadOnly:          true,
		},
	)
	require.Nil(t, err)

	clusterID, err := adminClient.GetClusterID(ctx)
	require.Nil(t, err)
	assert.Equal(t, "test-cluster-id", clusterID)

	_, err = NewClient(
		ctx,
		ClientConfig{
			ZKAddrs:           []string{util.TestZKAddr()},
			ZKPrefix:          clusterName,
			BootstrapAddrs:    []string{util.TestKafkaAddr()},
			ExpectedClusterID: "bad-cluster-id",
			ReadOnly:          true,
		},
	)
	require.NotNil(t, err)
}

func TestGetBrokers(t *testing.T) {
	zkConn, _, err := szk.Connect(
		[]string{util.TestZKAddr()},
		5*time.Second,
	)
	require.Nil(t, err)
	require.NotNil(t, zkConn)
	defer zkConn.Close()

	clusterName := testClusterID("brokers")
	zk.CreateNodes(
		t,
		zkConn,
		[]zk.PathTuple{
			{
				Path: fmt.Sprintf("/%s", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/brokers", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/brokers/ids", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/brokers/ids/1", clusterName),
				Obj: map[string]interface{}{
					"host":      "test1",
					"port":      1234,
					"rack":      "rack1",
					"timestamp": "1589603217000",
				},
			},
			{
				Path: fmt.Sprintf("/%s/brokers/ids/2", clusterName),
				Obj: map[string]interface{}{
					"host":      "test2",
					"port":      1234,
					"rack":      "rack2",
					"timestamp": "1589603217000",
				},
			},
			{
				Path: fmt.Sprintf("/%s/config", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/config/brokers", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/config/brokers/1", clusterName),
				Obj: map[string]interface{}{
					"version": 1,
					"config": map[string]string{
						"key1": "value1",
					},
				},
			},
		},
	)

	ctx := context.Background()
	adminClient, err := NewClient(
		ctx,
		ClientConfig{
			ZKAddrs:        []string{util.TestZKAddr()},
			ZKPrefix:       clusterName,
			BootstrapAddrs: []string{util.TestKafkaAddr()},
			ReadOnly:       true,
		},
	)
	require.Nil(t, err)
	defer adminClient.Close()

	brokers, err := adminClient.GetBrokers(ctx, nil)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(brokers))

	assert.Equal(
		t,
		BrokerInfo{
			ID:        1,
			Host:      "test1",
			Port:      1234,
			Rack:      "rack1",
			Timestamp: time.Unix(1589603217, 0),
			Config: map[string]string{
				"key1": "value1",
			},
		},
		brokers[0],
	)
	assert.Equal(
		t,
		BrokerInfo{
			ID:        2,
			Host:      "test2",
			Port:      1234,
			Rack:      "rack2",
			Timestamp: time.Unix(1589603217, 0),
		},
		brokers[1],
	)
}

func TestGetTopics(t *testing.T) {
	zkConn, _, err := szk.Connect(
		[]string{util.TestZKAddr()},
		5*time.Second,
	)
	require.Nil(t, err)
	require.NotNil(t, zkConn)
	defer zkConn.Close()

	clusterName := testClusterID("topics")
	zk.CreateNodes(
		t,
		zkConn,
		[]zk.PathTuple{
			{
				Path: fmt.Sprintf("/%s", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/brokers", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/brokers/topics", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/brokers/topics/topic1", clusterName),
				Obj: map[string]interface{}{
					"version": 1,
					"partitions": map[string][]int{
						"0": {1, 2},
						"1": {2, 3},
					},
				},
			},
			{
				Path: fmt.Sprintf("/%s/brokers/topics/topic1/partitions", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/brokers/topics/topic1/partitions/0", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/brokers/topics/topic1/partitions/0/state", clusterName),
				Obj: map[string]interface{}{
					"leader":           0,
					"version":          1,
					"isr":              []int{1, 2},
					"controller_epoch": 3,
					"leader_epoch":     5,
				},
			},
			{
				Path: fmt.Sprintf("/%s/brokers/topics/topic1/partitions/1", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/brokers/topics/topic1/partitions/1/state", clusterName),
				Obj: map[string]interface{}{
					"leader":           0,
					"version":          1,
					"isr":              []int{3, 2},
					"controller_epoch": 4,
					"leader_epoch":     6,
				},
			},
			{
				Path: fmt.Sprintf("/%s/brokers/topics/topic2", clusterName),
				Obj: map[string]interface{}{
					"version": 1,
					"partitions": map[string][]int{
						"0": {2},
					},
				},
			},
			{
				Path: fmt.Sprintf("/%s/brokers/topics/topic2/partitions", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/brokers/topics/topic2/partitions/0", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/brokers/topics/topic2/partitions/0/state", clusterName),
				Obj: map[string]interface{}{
					"leader":           0,
					"version":          1,
					"isr":              []int{2},
					"controller_epoch": 1,
					"leader_epoch":     2,
				},
			},
			{
				Path: fmt.Sprintf("/%s/config", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/config/topics", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/config/topics/topic1", clusterName),
				Obj: map[string]interface{}{
					"version": 0,
					"config": map[string]string{
						"key1": "value1",
					},
				},
			},
			{
				Path: fmt.Sprintf("/%s/config/topics/topic2", clusterName),
				Obj: map[string]interface{}{
					"version": 0,
					"config": map[string]string{
						"key2": "value2",
					},
				},
			},
		},
	)

	ctx := context.Background()
	adminClient, err := NewClient(
		ctx,
		ClientConfig{
			ZKAddrs:        []string{util.TestZKAddr()},
			ZKPrefix:       clusterName,
			BootstrapAddrs: []string{util.TestKafkaAddr()},
			ReadOnly:       true,
		},
	)
	require.Nil(t, err)
	defer adminClient.Close()

	topics, err := adminClient.GetTopics(ctx, nil, true)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(topics))
	assert.Equal(
		t,
		TopicInfo{
			Name: "topic1",
			Config: map[string]string{
				"key1": "value1",
			},
			Partitions: []PartitionInfo{
				{
					Topic:           "topic1",
					ID:              0,
					Leader:          0,
					Version:         1,
					Replicas:        []int{1, 2},
					ISR:             []int{1, 2},
					ControllerEpoch: 3,
					LeaderEpoch:     5,
				},
				{
					Topic:           "topic1",
					ID:              1,
					Leader:          0,
					Version:         1,
					Replicas:        []int{2, 3},
					ISR:             []int{3, 2},
					ControllerEpoch: 4,
					LeaderEpoch:     6,
				},
			},
			Version: 1,
		},
		topics[0],
	)
	assert.Equal(
		t,
		TopicInfo{
			Name: "topic2",
			Config: map[string]string{
				"key2": "value2",
			},
			Partitions: []PartitionInfo{
				{
					Topic:           "topic2",
					ID:              0,
					Leader:          0,
					Version:         1,
					Replicas:        []int{2},
					ISR:             []int{2},
					ControllerEpoch: 1,
					LeaderEpoch:     2,
				},
			},
			Version: 1,
		},
		topics[1],
	)

	topic1, err := adminClient.GetTopic(ctx, "topic1", true)
	assert.Nil(t, err)
	assert.Equal(
		t,
		TopicInfo{
			Name: "topic1",
			Config: map[string]string{
				"key1": "value1",
			},
			Partitions: []PartitionInfo{
				{
					Topic:           "topic1",
					ID:              0,
					Leader:          0,
					Version:         1,
					Replicas:        []int{1, 2},
					ISR:             []int{1, 2},
					ControllerEpoch: 3,
					LeaderEpoch:     5,
				},
				{
					Topic:           "topic1",
					ID:              1,
					Leader:          0,
					Version:         1,
					Replicas:        []int{2, 3},
					ISR:             []int{3, 2},
					ControllerEpoch: 4,
					LeaderEpoch:     6,
				},
			},
			Version: 1,
		},
		topic1,
	)

	_, err = adminClient.GetTopic(ctx, "non-existent-topic", true)
	assert.NotNil(t, err)
}

func TestGetBrokerPartitions(t *testing.T) {
	ctx := context.Background()
	adminClient, err := NewClient(
		ctx,
		ClientConfig{
			ZKAddrs:        []string{util.TestZKAddr()},
			BootstrapAddrs: []string{util.TestKafkaAddr()},
			ReadOnly:       false,
		},
	)
	require.Nil(t, err)
	defer adminClient.Close()

	topicName := util.RandomString("topic-create-", 6)

	config := kafka.TopicConfig{
		Topic:             topicName,
		NumPartitions:     2,
		ReplicationFactor: 2,
	}
	err = adminClient.CreateTopic(ctx, config)
	require.Nil(t, err)
	time.Sleep(250 * time.Millisecond)

	topicInfo, err := adminClient.getTopic(ctx, topicName, true)
	require.Nil(t, err)
	require.Equal(t, 2, len(topicInfo.Partitions))

	partitions, err := adminClient.GetBrokerPartitions(ctx, []string{topicName})
	require.Nil(t, err)
	require.Equal(t, 2, len(partitions))

	sort.Slice(partitions, func(a, b int) bool {
		return partitions[a].ID < partitions[b].ID
	})

	for p, partition := range partitions {
		assert.Equal(t, topicName, partition.Topic)
		assert.Equal(t, p, partition.ID)
		assert.Equal(t, topicInfo.Partitions[p].Leader, partition.Leader)
		assert.True(
			t,
			// Order returned by broker might not match what zk returns
			util.SameElements(partition.Replicas, topicInfo.Partitions[p].Replicas),
		)
		assert.True(
			t,
			// Order returned by broker might not match what zk returns
			util.SameElements(partition.ISR, topicInfo.Partitions[p].ISR),
		)
	}
}

func TestUpdateTopicConfig(t *testing.T) {
	zkConn, _, err := szk.Connect(
		[]string{util.TestZKAddr()},
		5*time.Second,
	)
	require.Nil(t, err)
	require.NotNil(t, zkConn)
	defer zkConn.Close()

	clusterName := testClusterID("topic-configs")
	zk.CreateNodes(
		t,
		zkConn,
		[]zk.PathTuple{
			{
				Path: fmt.Sprintf("/%s", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/config", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/config/changes", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/config/topics", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/config/topics/topic1", clusterName),
				Obj: map[string]interface{}{
					"version": 1,
					"config": map[string]string{
						"key1": "value1",
						"key2": "value2",
						"key4": "value4",
					},
				},
			},
		},
	)

	ctx := context.Background()
	adminClient, err := NewClient(
		ctx,
		ClientConfig{
			ZKAddrs:        []string{util.TestZKAddr()},
			ZKPrefix:       clusterName,
			BootstrapAddrs: []string{util.TestKafkaAddr()},
			ReadOnly:       false,
		},
	)
	require.Nil(t, err)
	defer adminClient.Close()

	updatedKeys, err := adminClient.UpdateTopicConfig(
		ctx,
		"topic1",
		[]kafka.ConfigEntry{
			{
				ConfigName:  "key2",
				ConfigValue: "value2-updated",
			},
			{
				ConfigName:  "key3",
				ConfigValue: "value3",
			},
			// Remove this key
			{
				ConfigName:  "key4",
				ConfigValue: "",
			},
		},
		true,
	)
	require.Nil(t, err)
	assert.Equal(
		t,
		[]string{
			"key2",
			"key3",
			"key4",
		},
		updatedKeys,
	)

	updatedKeys, err = adminClient.UpdateTopicConfig(
		ctx,
		"topic1",
		[]kafka.ConfigEntry{
			{
				ConfigName:  "key2",
				ConfigValue: "value2-updated2",
			},
			{
				ConfigName:  "key3",
				ConfigValue: "value3-updated",
			},
			{
				ConfigName:  "key5",
				ConfigValue: "new-value",
			},
		},
		false,
	)
	require.Nil(t, err)
	assert.Equal(
		t,
		[]string{
			"key5",
		},
		updatedKeys,
	)

	updatedConfig, _, err := adminClient.zkClient.Get(
		ctx,
		fmt.Sprintf("/%s/config/topics/topic1", clusterName),
	)
	assert.Nil(t, err)
	assert.JSONEq(
		t,
		`{"config":{"key1":"value1","key2":"value2-updated","key3":"value3","key5":"new-value"},"version":1}`,
		string(updatedConfig),
	)

	changes, _, err := adminClient.zkClient.Children(
		ctx,
		fmt.Sprintf("/%s/config/changes", clusterName),
	)
	assert.Nil(t, err)
	assert.Greater(t, len(changes), 0)

	change, _, err := adminClient.zkClient.Get(
		ctx,
		fmt.Sprintf("/%s/config/changes/%s", clusterName, changes[len(changes)-1]),
	)
	assert.Nil(t, err)
	assert.JSONEq(
		t,
		`{"entity_path":"topics/topic1","version":2}`,
		string(change),
	)
}

func TestUpdateBrokerConfig(t *testing.T) {
	zkConn, _, err := szk.Connect(
		[]string{util.TestZKAddr()},
		5*time.Second,
	)
	require.Nil(t, err)
	require.NotNil(t, zkConn)
	defer zkConn.Close()

	clusterName := testClusterID("broker-configs")
	zk.CreateNodes(
		t,
		zkConn,
		[]zk.PathTuple{
			{
				Path: fmt.Sprintf("/%s", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/config", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/config/changes", clusterName),
				Obj:  nil,
			},
			// The /config/brokers path will be created automatically.
		},
	)

	ctx := context.Background()
	adminClient, err := NewClient(
		ctx,
		ClientConfig{
			ZKAddrs:        []string{util.TestZKAddr()},
			ZKPrefix:       clusterName,
			BootstrapAddrs: []string{util.TestKafkaAddr()},
			ReadOnly:       false,
		},
	)
	require.Nil(t, err)
	defer adminClient.Close()

	updatedKeys, err := adminClient.UpdateBrokerConfig(
		ctx,
		1,
		[]kafka.ConfigEntry{
			{
				ConfigName:  "key2",
				ConfigValue: "value2-updated",
			},
			{
				ConfigName:  "key3",
				ConfigValue: "value3",
			},
		},
		true,
	)
	require.Nil(t, err)
	assert.Equal(
		t,
		[]string{
			"key2",
			"key3",
		},
		updatedKeys,
	)

	updatedKeys, err = adminClient.UpdateBrokerConfig(
		ctx,
		1,
		[]kafka.ConfigEntry{
			{
				ConfigName:  "key2",
				ConfigValue: "value2-updated2",
			},
			{
				ConfigName:  "key3",
				ConfigValue: "",
			},
			{
				ConfigName:  "key5",
				ConfigValue: "new-value",
			},
		},
		false,
	)
	require.Nil(t, err)
	// Only key 5 is updated because overwrite is set to false
	assert.Equal(
		t,
		[]string{
			"key5",
		},
		updatedKeys,
	)

	updatedConfig, _, err := adminClient.zkClient.Get(
		ctx,
		fmt.Sprintf("/%s/config/brokers/1", clusterName),
	)
	assert.Nil(t, err)
	assert.JSONEq(
		t,
		`{"config":{"key2":"value2-updated","key3":"value3","key5":"new-value"},"version":1}`,
		string(updatedConfig),
	)

	changes, _, err := adminClient.zkClient.Children(
		ctx,
		fmt.Sprintf("/%s/config/changes", clusterName),
	)
	assert.Nil(t, err)
	assert.Greater(t, len(changes), 0)

	change, _, err := adminClient.zkClient.Get(
		ctx,
		fmt.Sprintf("/%s/config/changes/%s", clusterName, changes[len(changes)-1]),
	)
	assert.Nil(t, err)
	assert.JSONEq(
		t,
		`{"entity_path":"brokers/1","version":2}`,
		string(change),
	)
}

func TestCreateTopic(t *testing.T) {
	ctx := context.Background()
	adminClient, err := NewClient(
		ctx,
		ClientConfig{
			ZKAddrs:        []string{util.TestZKAddr()},
			ZKPrefix:       "",
			BootstrapAddrs: []string{util.TestKafkaAddr()},
			ReadOnly:       false,
		},
	)
	require.Nil(t, err)
	defer adminClient.Close()

	topicName := util.RandomString("topic-create-", 6)

	config := kafka.TopicConfig{
		Topic:             topicName,
		NumPartitions:     2,
		ReplicationFactor: 2,
	}
	err = adminClient.CreateTopic(ctx, config)
	require.Nil(t, err)
	time.Sleep(250 * time.Millisecond)

	topics, err := adminClient.GetTopics(ctx, []string{topicName}, false)
	require.Nil(t, err)
	require.Equal(t, 1, len(topics))
	assert.Equal(t, topicName, topics[0].Name)
}

func TestUpdateAssignments(t *testing.T) {
	zkConn, _, err := szk.Connect(
		[]string{util.TestZKAddr()},
		5*time.Second,
	)
	require.Nil(t, err)
	require.NotNil(t, zkConn)
	defer zkConn.Close()

	clusterName := testClusterID("assignments")
	zk.CreateNodes(
		t,
		zkConn,
		[]zk.PathTuple{
			{
				Path: fmt.Sprintf("/%s", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/admin", clusterName),
				Obj:  nil,
			},
		},
	)

	ctx := context.Background()
	adminClient, err := NewClient(
		ctx,
		ClientConfig{
			ZKAddrs:        []string{util.TestZKAddr()},
			ZKPrefix:       clusterName,
			BootstrapAddrs: []string{util.TestKafkaAddr()},
			ReadOnly:       false,
		},
	)
	require.Nil(t, err)
	defer adminClient.Close()

	exists, err := adminClient.AssignmentInProgress(ctx)
	assert.Nil(t, err)
	assert.False(t, exists)

	err = adminClient.AssignPartitions(
		ctx,
		"test-topic",
		[]PartitionAssignment{
			{
				ID:       1,
				Replicas: []int{1, 2, 3},
			},
			{
				ID:       2,
				Replicas: []int{3, 4, 5},
			},
		},
	)
	require.Nil(t, err)

	reassignment, _, err := adminClient.zkClient.Get(
		ctx,
		fmt.Sprintf("/%s/admin/reassign_partitions", clusterName),
	)
	require.Nil(t, err)
	assert.JSONEq(
		t,
		`{
			"version":1,
			"partitions": [
				{"topic":"test-topic","partition":1,"replicas":[1,2,3]},
				{"topic":"test-topic","partition":2,"replicas":[3,4,5]}
			]
		}`,
		string(reassignment),
	)

	exists, err = adminClient.AssignmentInProgress(ctx)
	assert.Nil(t, err)
	assert.True(t, exists)
}

func TestAddPartitions(t *testing.T) {
	zkConn, _, err := szk.Connect(
		[]string{util.TestZKAddr()},
		5*time.Second,
	)
	require.Nil(t, err)
	require.NotNil(t, zkConn)
	defer zkConn.Close()

	clusterName := testClusterID("add-partitions")
	zk.CreateNodes(
		t,
		zkConn,
		[]zk.PathTuple{
			{
				Path: fmt.Sprintf("/%s", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/brokers", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/brokers/topics", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/brokers/topics/topic1", clusterName),
				Obj: map[string]interface{}{
					"version": 1,
					"partitions": map[string][]int{
						"0": {1, 2},
						"1": {2, 3},
					},
				},
			},
			{
				Path: fmt.Sprintf("/%s/config", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/config/topics", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/config/topics/topic1", clusterName),
				Obj: map[string]interface{}{
					"version": 1,
				},
			},
		},
	)

	ctx := context.Background()
	adminClient, err := NewClient(
		ctx,
		ClientConfig{
			ZKAddrs:        []string{util.TestZKAddr()},
			ZKPrefix:       clusterName,
			BootstrapAddrs: []string{util.TestKafkaAddr()},
			ReadOnly:       false,
		},
	)
	require.Nil(t, err)
	defer adminClient.Close()

	exists, err := adminClient.AssignmentInProgress(ctx)
	assert.Nil(t, err)
	assert.False(t, exists)

	err = adminClient.AddPartitions(
		ctx,
		"topic1",
		[]PartitionAssignment{
			{
				ID:       2,
				Replicas: []int{1, 2},
			},
			{
				ID:       3,
				Replicas: []int{3, 4},
			},
		},
	)
	require.Nil(t, err)

	topicInfo, err := adminClient.getTopic(ctx, "topic1", false)
	require.Nil(t, err)
	assert.Equal(
		t,
		[]PartitionInfo{
			{
				Topic:    "topic1",
				ID:       0,
				Replicas: []int{1, 2},
			},
			{
				Topic:    "topic1",
				ID:       1,
				Replicas: []int{2, 3},
			},
			{
				Topic:    "topic1",
				ID:       2,
				Replicas: []int{1, 2},
			},
			{
				Topic:    "topic1",
				ID:       3,
				Replicas: []int{3, 4},
			},
		},
		topicInfo.Partitions,
	)

	// Adding a partition that exists leads to an error
	err = adminClient.AddPartitions(
		ctx,
		"topic1",
		[]PartitionAssignment{
			{
				ID:       3,
				Replicas: []int{1, 2},
			},
			{
				ID:       4,
				Replicas: []int{3, 4},
			},
		},
	)
	require.NotNil(t, err)
}

func TestRunLeaderElection(t *testing.T) {
	zkConn, _, err := szk.Connect(
		[]string{util.TestZKAddr()},
		5*time.Second,
	)
	require.Nil(t, err)
	require.NotNil(t, zkConn)
	defer zkConn.Close()

	clusterName := testClusterID("elections")
	zk.CreateNodes(
		t,
		zkConn,
		[]zk.PathTuple{
			{
				Path: fmt.Sprintf("/%s", clusterName),
				Obj:  nil,
			},
			{
				Path: fmt.Sprintf("/%s/admin", clusterName),
				Obj:  nil,
			},
		},
	)

	ctx := context.Background()
	adminClient, err := NewClient(
		ctx,
		ClientConfig{
			ZKAddrs:        []string{util.TestZKAddr()},
			ZKPrefix:       clusterName,
			BootstrapAddrs: []string{util.TestKafkaAddr()},
			ReadOnly:       false,
		},
	)
	require.Nil(t, err)
	defer adminClient.Close()

	exists, err := adminClient.ElectionInProgress(ctx)
	assert.Nil(t, err)
	assert.False(t, exists)

	err = adminClient.RunLeaderElection(
		ctx,
		"test-topic",
		[]int{3, 5, 6},
	)
	require.Nil(t, err)

	reassignment, _, err := adminClient.zkClient.Get(
		ctx,
		fmt.Sprintf("/%s/admin/preferred_replica_election", clusterName),
	)
	require.Nil(t, err)
	assert.JSONEq(
		t,
		`{
			"version":1,
			"partitions": [
				{"topic":"test-topic", "partition":3},
				{"topic":"test-topic", "partition":5},
				{"topic":"test-topic", "partition":6}
			]
		}`,
		string(reassignment),
	)

	exists, err = adminClient.ElectionInProgress(ctx)
	assert.Nil(t, err)
	assert.True(t, exists)
}

func testLocking(t *testing.T) {
	ctx := context.Background()
	adminClient, err := NewClient(
		ctx,
		ClientConfig{
			ZKAddrs:        []string{util.TestZKAddr()},
			BootstrapAddrs: []string{util.TestKafkaAddr()},
		},
	)
	require.Nil(t, err)
	defer adminClient.Close()

	lockPath := fmt.Sprintf("/locks/%s", util.RandomString("", 8))
	held, err := adminClient.LockHeld(ctx, lockPath)
	assert.Nil(t, err)
	assert.False(t, held)

	lock, err := adminClient.AcquireLock(ctx, lockPath)
	require.Nil(t, err)

	held, err = adminClient.LockHeld(ctx, lockPath)
	assert.Nil(t, err)
	assert.True(t, held)

	lock.Unlock()
	held, err = adminClient.LockHeld(ctx, lockPath)
	assert.Nil(t, err)
	assert.False(t, held)
}

func testClusterID(name string) string {
	return util.RandomString(fmt.Sprintf("cluster-%s-", name), 6)
}
