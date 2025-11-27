package migrations

import (
	"livo-backend/models"
	"livo-backend/utilities"
	"log"

	"gorm.io/gorm"
)

// AutoMigrate runs database migrations
func AutoMigrate(db *gorm.DB) {
	// Run migrations
	err := db.AutoMigrate(
		&models.Role{},
		&models.User{},
		&models.UserRole{},
	)
	if err != nil {
		log.Printf("⚠️ Warning: Failed to migrate some tables: %v", err)
	} else {
		log.Println("✓ Database migration completed successfully")
	}

	// Seed default roles
	seedDefaultRoles(db)

	// Seed first superadmin user
	seedSuperadminUser(db)

	// Seed default boxes
	// seedDefaultBoxes(db)

	// Seed default channels
	// seedDefaultChannels(db)

	// Seed default expeditions
	// seedDefaultExpeditions(db)

	// Seed default stores
	// seedDefaultStores(db)
}

// seedDefaultRoles creates default roles if they don't exist
func seedDefaultRoles(db *gorm.DB) {
	roles := []models.Role{
		{Name: "superadmin", Description: "Super Administrator with full system access"},
		{Name: "coordinator", Description: "Coordinator with high-level management access"},
		{Name: "admin", Description: "Administrator with mid-level management access"},
		{Name: "retur", Description: "Retur with mid-level management access"},
		{Name: "finance", Description: "Finance role with financial management access"},
		{Name: "picker", Description: "Picker with basic operational access"},
		{Name: "outbound", Description: "Outbound role with shipping management access"},
		{Name: "qc-ribbon", Description: "Quality Control for Ribbon products"},
		{Name: "qc-online", Description: "Quality Control for Online products"},
		{Name: "guest", Description: "Guest with limited access"},
	}

	for _, role := range roles {
		var existingRole models.Role
		if err := db.Where("name = ?", role.Name).First(&existingRole).Error; err != nil {
			// Role doesn't exist, create it
			if err := db.Create(&role).Error; err != nil {
				log.Printf("Failed to create role %s: %v", role.Name, err)
			} else {
				log.Printf("Created role: %s", role.Name)
			}
		}
	}
}

// Seed default store data
// func seedDefaultStores(db *gorm.DB) {
// 	stores := []models.Store{
// 		{Code: "AX", Name: "Axon"},
// 		{Code: "DR", Name: "DeParcel Ribbon"},
// 		{Code: "AS", Name: "Axon Store"},
// 		{Code: "AL", Name: "Aqualivo"},
// 		{Code: "LM", Name: "Livo Mall"},
// 		{Code: "LI", Name: "Livo ID"},
// 		{Code: "BI", Name: "Bion"},
// 		{Code: "AI", Name: "Axon ID"},
// 		{Code: "AM", Name: "Axon Mall"},
// 		{Code: "AS", Name: "Aqualivo Store"},
// 		{Code: "RP", Name: "Rumah Pita"},
// 		{Code: "SL", Name: "Sporti Livo"},
// 		{Code: "LT", Name: "Livotech"},
// 	}

// 	for _, store := range stores {
// 		var existingStore models.Store
// 		if err := db.Where("code = ?", store.Code).First(&existingStore).Error; err != nil {
// 			// Store doesn't exist, create it
// 			if err := db.Create(&store).Error; err != nil {
// 				log.Printf("Failed to create store %s: %v", store.Name, err)
// 			} else {
// 				log.Printf("Created store: %s", store.Name)
// 			}
// 		}
// 	}
// }

// Seed default expedition data
// func seedDefaultExpeditions(db *gorm.DB) {
// 	expeditions := []models.Expedition{
// 		{Code: "TKP0", Name: "JNE/ID-Express", Slug: "jne-id-express", Color: "#006072"}, // JNE/ID Express
// 		{Code: "PJ", Name: "Offline", Slug: "offline", Color: "#000000"},                 // Offline
// 		{Code: "INS", Name: "Instant", Slug: "instant", Color: "#00d0dd"},                // Instant
// 		{Code: "BLMP", Name: "Paxel", Slug: "paxel", Color: "#5f50a0"},                   // Paxel
// 		{Code: "LX", Name: "LEX", Slug: "lex", Color: "#0c5eb4"},                         // LEX
// 		{Code: "NL", Name: "LEX", Slug: "lex", Color: "#0c5eb4"},                         // LEX
// 		{Code: "JN", Name: "LEX", Slug: "lex", Color: "#0c5eb4"},                         // LEX
// 		{Code: "JZ", Name: "LEX", Slug: "lex", Color: "#0c5eb4"},                         // LEX
// 		{Code: "SP", Name: "SPX", Slug: "spx", Color: "#ff7300"},                         // SPX
// 		{Code: "ID2", Name: "SPX", Slug: "spx", Color: "#ff7300"},                        // SPX
// 		{Code: "TSA", Name: "AnterAja", Slug: "anteraja", Color: "#ff007a"},              // AnterAja
// 		{Code: "1100", Name: "AnterAja", Slug: "anteraja", Color: "#ff007a"},             // AnterAja
// 		{Code: "TAA", Name: "AnterAja", Slug: "anteraja", Color: "#ff007a"},              // AnterAja
// 		{Code: "TLJX", Name: "JNE", Slug: "jne", Color: "#032078"},                       // JNE
// 		{Code: "41", Name: "JNE", Slug: "jne", Color: "#032078"},                         // JNE
// 		{Code: "CM", Name: "JNE", Slug: "jne", Color: "#032078"},                         // JNE
// 		{Code: "BLIJ", Name: "JNE", Slug: "jne", Color: "#032078"},                       // JNE
// 		{Code: "JT", Name: "JNE", Slug: "jne", Color: "#032078"},                         // JNE
// 		{Code: "TG", Name: "JNE", Slug: "jne", Color: "#032078"},                         // JNE
// 		{Code: "TLJR", Name: "JNE", Slug: "jne", Color: "#032078"},                       // JNE
// 		{Code: "TLJC", Name: "JNE", Slug: "jne", Color: "#032078"},                       // JNE
// 		{Code: "JNE", Name: "JNE", Slug: "jne", Color: "#032078"},                        // JNE
// 		{Code: "JO", Name: "J&T Express", Slug: "j&t-express", Color: "#ff0000"},         // J&T Express
// 		{Code: "JD", Name: "J&T Express", Slug: "j&t-express", Color: "#ff0000"},         // J&T Express
// 		{Code: "JJ", Name: "J&T Express", Slug: "j&t-express", Color: "#ff0000"},         // J&T Express
// 		{Code: "JB", Name: "J&T Express", Slug: "j&t-express", Color: "#ff0000"},         // J&T Express
// 		{Code: "JP", Name: "J&T Express", Slug: "j&t-express", Color: "#ff0000"},         // J&T Express
// 		{Code: "JX", Name: "J&T Express", Slug: "j&t-express", Color: "#ff0000"},         // J&T Express
// 		{Code: "TKJN", Name: "J&T Express", Slug: "j&t-express", Color: "#ff0000"},       // J&T Express
// 		{Code: "IDS", Name: "ID Express", Slug: "id-express", Color: "#b30000"},          // ID Express
// 		{Code: "TKP8", Name: "ID Express", Slug: "id-express", Color: "#b30000"},         // ID Express
// 		{Code: "300", Name: "J&T Cargo", Slug: "j&t-cargo", Color: "#008601"},            // J&T Cargo
// 		{Code: "2012", Name: "J&T Cargo", Slug: "j&t-cargo", Color: "#008601"},           // J&T Cargo
// 		{Code: "2011", Name: "J&T Cargo", Slug: "j&t-cargo", Color: "#008601"},           // J&T Cargo
// 		{Code: "2010", Name: "J&T Cargo", Slug: "j&t-cargo", Color: "#008601"},           // J&T Cargo
// 		{Code: "2009", Name: "J&T Cargo", Slug: "j&t-cargo", Color: "#008601"},           // J&T Cargo
// 		{Code: "2008", Name: "J&T Cargo", Slug: "j&t-cargo", Color: "#008601"},           // J&T Cargo
// 		{Code: "2007", Name: "J&T Cargo", Slug: "j&t-cargo", Color: "#008601"},           // J&T Cargo
// 		{Code: "2006", Name: "J&T Cargo", Slug: "j&t-cargo", Color: "#008601"},           // J&T Cargo
// 		{Code: "2005", Name: "J&T Cargo", Slug: "j&t-cargo", Color: "#008601"},           // J&T Cargo
// 		{Code: "TS", Name: "Wahana", Slug: "wahana", Color: "#ffa100"},                   // Wahana
// 		{Code: "SIC", Name: "SiCepat", Slug: "sicepat", Color: "#830000"},                // SiCepat
// 	}

// 	for _, expedition := range expeditions {
// 		var existingExpedition models.Expedition
// 		if err := db.Where("code = ?", expedition.Code).First(&existingExpedition).Error; err != nil {
// 			// Expedition doesn't exist, create it
// 			if err := db.Create(&expedition).Error; err != nil {
// 				log.Printf("Failed to create expedition %s: %v", expedition.Code, err)
// 			} else {
// 				log.Printf("Created expedition: %s", expedition.Code)
// 			}
// 		}
// 	}
// }

// Seed default channel data
// func seedDefaultChannels(db *gorm.DB) {
// 	channels := []models.Channel{
// 		{Code: "SP", Name: "Shopee"},
// 		{Code: "TP", Name: "Tokopedia"},
// 		{Code: "LA", Name: "Lazada"},
// 		{Code: "BU", Name: "Bukalapak"},
// 		{Code: "BL", Name: "Blibli"},
// 		{Code: "TT", Name: "Tiktok"},
// 	}

// 	for _, channel := range channels {
// 		var existingChannel models.Channel
// 		if err := db.Where("code = ?", channel.Code).First(&existingChannel).Error; err != nil {
// 			// Channel doesn't exist, create it
// 			if err := db.Create(&channel).Error; err != nil {
// 				log.Printf("Failed to create channel %s: %v", channel.Name, err)
// 			} else {
// 				log.Printf("Created channel: %s", channel.Name)
// 			}
// 		}
// 	}
// }

// Seed default box data
// func seedDefaultBoxes(db *gorm.DB) {
// 	boxes := []models.Box{
// 		{Code: "PC", Name: "Packing"},
// 		{Code: "1", Name: "001"},
// 		{Code: "2", Name: "002"},
// 		{Code: "A", Name: "Polos A"},
// 		{Code: "B", Name: "Polos B"},
// 		{Code: "K", Name: "Kawat"},
// 		{Code: "R", Name: "Ribbon"},
// 		{Code: "PK", Name: "Panjang Kecil"},
// 		{Code: "PB", Name: "Panjang Besar"},
// 		{Code: "SF", Name: "Single Face"},
// 		{Code: "L", Name: "Layer"},
// 		{Code: "X", Name: "Dos Bekas"},
// 		{Code: "KRG", Name: "Karung"},
// 		{Code: "17", Name: "1730"},
// 		{Code: "20", Name: "2030"},
// 		{Code: "25", Name: "2535"},
// 		{Code: "30", Name: "3040"},
// 		{Code: "35", Name: "3550"},
// 		{Code: "40", Name: "4050"},
// 		{Code: "75", Name: "5075"},
// 		{Code: "85", Name: "8525"},
// 		{Code: "70", Name: "7020"},
// 		{Code: "50", Name: "6050"},
// 		{Code: "KR", Name: "Kantong Kresek"},
// 	}

// 	for _, box := range boxes {
// 		var existingBox models.Box
// 		if err := db.Where("code = ?", box.Code).First(&existingBox).Error; err != nil {
// 			// Box doesn't exist, create it
// 			if err := db.Create(&box).Error; err != nil {
// 				log.Printf("Failed to create box %s: %v", box.Name, err)
// 			} else {
// 				log.Printf("Created box: %s", box.Name)
// 			}
// 		}
// 	}
// }

// seedSuperadminUser creates the first superadmin user if it doesn't exist
func seedSuperadminUser(db *gorm.DB) {
	// Check if superadmin user already exists
	var existingUser models.User
	if err := db.Where("username = ?", "superadmin").First(&existingUser).Error; err == nil {
		log.Println("Superadmin user already exists")
		return
	}

	// Hash password
	hashedPassword, err := utilities.HashPassword("123456")
	if err != nil {
		log.Printf("Failed to hash superadmin password: %v", err)
		return
	}

	// Create superadmin user
	user := models.User{
		Username: "superadmin",
		Email:    "superadmin@example.com",
		Password: hashedPassword,
		FullName: "Super Administrator Sepertinya",
		IsActive: true,
	}

	if err := db.Create(&user).Error; err != nil {
		log.Printf("Failed to create superadmin user: %v", err)
		return
	}

	// Find superadmin role
	var superadminRole models.Role
	if err := db.Where("name = ?", "superadmin").First(&superadminRole).Error; err != nil {
		log.Printf("Superadmin role not found: %v", err)
		return
	}

	// Assign superadmin role
	userRole := models.UserRole{
		UserID:     user.ID,
		RoleID:     superadminRole.ID,
		AssignedBy: user.ID, // Self-assigned for the first superadmin
	}

	if err := db.Create(&userRole).Error; err != nil {
		log.Printf("Failed to assign superadmin role: %v", err)
		return
	}

	log.Println("✓ Superadmin user created successfully (username: superadmin, password: 123456)")
}
