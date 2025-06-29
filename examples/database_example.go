package main

import (
	"fmt"
	"log"
	"time"

	"github.com/ysicing/nexus/pkg/cluster"
	"github.com/ysicing/nexus/pkg/database"
)

func main() {
	fmt.Println("=== Nexus 数据库集成示例 ===")

	// 1. 创建数据库配置（使用 DSN 方式）
	fmt.Println("\n1. 初始化数据库配置...")
	dbConfig := database.GetDefaultConfig()
	fmt.Printf("数据库 DSN: %s\n", dbConfig.DSN)

	// 2. 初始化数据库
	fmt.Println("\n2. 初始化数据库连接...")
	db, err := database.NewDatabase(dbConfig)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer db.Close()

	// 3. 执行数据库迁移
	fmt.Println("\n3. 执行数据库迁移...")
	if err := db.MigrateDatabase(); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 4. 创建带数据库支持的集群管理器
	fmt.Println("\n4. 初始化集群管理器...")
	clusterManager := cluster.NewManagerWithDB(db)
	if err := clusterManager.Initialize(); err != nil {
		log.Fatalf("初始化集群管理器失败: %v", err)
	}
	defer clusterManager.Stop()

	// 5. 展示集群加载过程
	fmt.Println("\n5. 集群加载结果:")
	clusters := clusterManager.ListClusters()
	fmt.Printf("总共加载了 %d 个集群:\n", len(clusters))

	for i, cluster := range clusters {
		fmt.Printf("  [%d] %s (%s)\n", i+1, cluster.Name, cluster.ID)
		fmt.Printf("      服务器: %s\n", cluster.Server)
		fmt.Printf("      状态: %s\n", cluster.Status)
		if cluster.IsDefault {
			fmt.Printf("      ✓ 默认集群\n")
		}
		if len(cluster.Labels) > 0 {
			fmt.Printf("      标签: %v\n", cluster.Labels)
		}
		fmt.Println()
	}

	// 6. 获取默认集群
	fmt.Println("6. 默认集群信息:")
	defaultCluster, err := clusterManager.GetDefaultCluster()
	if err != nil {
		fmt.Printf("   没有找到默认集群: %v\n", err)
	} else {
		fmt.Printf("   默认集群: %s (%s)\n", defaultCluster.Name, defaultCluster.ID)
		fmt.Printf("   服务器: %s\n", defaultCluster.Server)
		fmt.Printf("   版本: %s\n", defaultCluster.Version)
	}

	// 7. 演示添加新集群
	fmt.Println("\n7. 演示添加新集群...")
	labels := map[string]string{
		"environment": "demo",
		"region":      "local",
	}

	newCluster, err := clusterManager.AddCluster(
		"演示集群",
		"这是一个演示用的集群配置",
		"", // 空的 kubeconfig 内容
		labels,
	)
	if err != nil {
		fmt.Printf("   添加集群失败: %v\n", err)
	} else {
		fmt.Printf("   成功添加集群: %s (%s)\n", newCluster.Name, newCluster.ID)
	}

	// 8. 再次列出集群
	fmt.Println("\n8. 更新后的集群列表:")
	clusters = clusterManager.ListClusters()
	fmt.Printf("总共 %d 个集群:\n", len(clusters))
	for i, cluster := range clusters {
		fmt.Printf("  [%d] %s (%s)\n", i+1, cluster.Name, cluster.ID)
		if cluster.IsDefault {
			fmt.Printf("      ✓ 默认集群\n")
		}
	}

	// 9. 演示数据库持久化
	fmt.Println("\n9. 数据库持久化测试:")
	fmt.Println("   集群信息已保存到数据库")
	fmt.Println("   重启应用后，集群配置将自动恢复")

	// 10. 等待一段时间展示健康检查
	fmt.Println("\n10. 健康检查演示 (等待 5 秒)...")
	time.Sleep(5 * time.Second)

	fmt.Println("\n=== 示例完成 ===")
	fmt.Println("数据库集成功能已成功演示!")
	fmt.Println("\n支持的 DSN 格式:")
	fmt.Println("  SQLite:     sqlite:./data/nexus.db")
	fmt.Println("  MySQL:      mysql://user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local")
	fmt.Println("  PostgreSQL: postgres://user:password@localhost:5432/dbname?sslmode=disable")
}
