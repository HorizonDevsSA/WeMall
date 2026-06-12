package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CategorySeed struct {
	NameEn     string
	NameSn     string
	NameNd     string
	Schema     map[string]interface{}
	Subcategories []CategorySeed
}

func main() {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://wemall:wemall_secret@localhost:5433/wemall_products?sslmode=disable"
	}

	fmt.Println("Connecting to product database...")
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Printf("Database connection failed: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		fmt.Printf("Database ping failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Cleaning existing categories...")
	_, _ = pool.Exec(ctx, "TRUNCATE categories CASCADE")

	// Define categories with schemas
	clothingSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"size": map[string]interface{}{
				"type": "string",
				"enum": []string{"XS", "S", "M", "L", "XL", "XXL", "3XL"},
			},
			"color": map[string]interface{}{"type": "string"},
			"material": map[string]interface{}{"type": "string"},
			"season": map[string]interface{}{"type": "string"},
		},
		"required": []string{"size", "color"},
	}

	shoesSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"size": map[string]interface{}{
				"type": "string",
			},
			"color": map[string]interface{}{"type": "string"},
			"material": map[string]interface{}{"type": "string"},
		},
		"required": []string{"size", "color"},
	}

	phoneSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"brand": map[string]interface{}{"type": "string"},
			"os": map[string]interface{}{"type": "string"},
			"storage": map[string]interface{}{
				"type": "string",
				"enum": []string{"64GB", "128GB", "256GB", "512GB", "1TB"},
			},
			"ram": map[string]interface{}{
				"type": "string",
				"enum": []string{"4GB", "6GB", "8GB", "12GB", "16GB"},
			},
		},
		"required": []string{"brand", "storage", "ram"},
	}

	beautySchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"skin_type": map[string]interface{}{
				"type": "string",
				"enum": []string{"dry", "oily", "combination", "sensitive", "all"},
			},
			"product_type": map[string]interface{}{"type": "string"},
		},
		"required": []string{"skin_type", "product_type"},
	}

	furnitureSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"material": map[string]interface{}{"type": "string"},
			"color":    map[string]interface{}{"type": "string"},
			"dimensions": map[string]interface{}{"type": "string"},
		},
		"required": []string{"material", "color"},
	}

	foodSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"weight_g":   map[string]interface{}{"type": "integer"},
			"volume_ml":  map[string]interface{}{"type": "integer"},
			"expiry_days": map[string]interface{}{"type": "integer"},
		},
	}

	categories := []CategorySeed{
		{
			NameEn: "Fashion & Apparel",
			NameSn: "Fashoni neZvipfeko",
			NameNd: "Fashoni neZokugqoka",
			Subcategories: []CategorySeed{
				{
					NameEn: "Women's Clothing",
					NameSn: "Zvipfeko zveMadzimai",
					NameNd: "Izokugqoka zeSifazane",
					Schema: clothingSchema,
					Subcategories: []CategorySeed{
						{NameEn: "Dresses", NameSn: "Dhirezi", NameNd: "Amaloko"},
						{NameEn: "Tops & Blouses", NameSn: "Tops neMabhurawuzi", NameNd: "Amathops"},
						{NameEn: "Skirts & Pants", NameSn: "Siketi neMatirauzi", NameNd: "Iziketi leMabhurugwa"},
						{NameEn: "Jackets & Coats", NameSn: "Jackets neMakoti", NameNd: "Ajasi"},
					},
				},
				{
					NameEn: "Men's Clothing",
					NameSn: "Zvipfeko zveVarume",
					NameNd: "Izokugqoka zeSilisa",
					Schema: clothingSchema,
					Subcategories: []CategorySeed{
						{NameEn: "Shirts & T-Shirts", NameSn: "Shirts neT-Shirts", NameNd: "Amashati leZikipa"},
						{NameEn: "Jeans & Trousers", NameSn: "Jeans neMatirauzi", NameNd: "Amabhurugwa"},
						{NameEn: "Suits & Blazers", NameSn: "Masutu", NameNd: "Amasudu"},
					},
				},
				{
					NameEn: "Shoes & Footwear",
					NameSn: "Shangu",
					NameNd: "Izicathulo",
					Schema: shoesSchema,
					Subcategories: []CategorySeed{
						{NameEn: "Sneakers & Athletic", NameSn: "Sneakers", NameNd: "Izicathulo zemidlalo"},
						{NameEn: "Formal Shoes", NameSn: "Shangu dzeHofisi", NameNd: "Izicathulo zeOfisi"},
						{NameEn: "Heels & Sandals", NameSn: "Shangu dzeMadzimai", NameNd: "Izicathulo zeSifazane"},
					},
				},
				{
					NameEn: "Fashion Accessories",
					NameSn: "Zvishongo zveFashoni",
					NameNd: "Izihlobiso zeFashoni",
					Subcategories: []CategorySeed{
						{NameEn: "Sunglasses", NameSn: "Magirazi ezuva", NameNd: "Amaglasi elanga"},
						{NameEn: "Belts", NameSn: "Mabhande", NameNd: "Amabhande"},
						{NameEn: "Hats & Caps", NameSn: "Nguwani", NameNd: "Izihko"},
					},
				},
			},
		},
		{
			NameEn: "Electronics & Technology",
			NameSn: "ZveMagetsi neTekinoroji",
			NameNd: "Ezomanyalo neTeknoloji",
			Subcategories: []CategorySeed{
				{
					NameEn: "Smartphones & Tablets",
					NameSn: "Nharembozha",
					NameNd: "Ucingo lweSandla",
					Schema: phoneSchema,
					Subcategories: []CategorySeed{
						{NameEn: "iPhones", NameSn: "MaApple Phones", NameNd: "Ifoni zeApple"},
						{NameEn: "Android Phones", NameSn: "MaAndroid Phones", NameNd: "Ifoni zeAndroid"},
						{NameEn: "Tablets & iPads", NameSn: "Matablets", NameNd: "Amatablet"},
					},
				},
				{
					NameEn: "Computers & Laptops",
					NameSn: "Makombiyuta",
					NameNd: "Amakhompyutha",
					Subcategories: []CategorySeed{
						{NameEn: "Laptops", NameSn: "Malaptops", NameNd: "Amalaptop"},
						{NameEn: "Desktop PCs", NameSn: "Makombiyuta emuHofisi", NameNd: "Amakhompyutha"},
						{NameEn: "Computer Monitors", NameSn: "Mamonita", NameNd: "Amamonitha"},
					},
				},
				{
					NameEn: "Audio & Entertainment",
					NameSn: "ZvekuTeereresa",
					NameNd: "Izinto zokulalela",
					Subcategories: []CategorySeed{
						{NameEn: "Headphones & Earbuds", NameSn: "MaHeadphones", NameNd: "Izinto zokulalela"},
						{NameEn: "Bluetooth Speakers", NameSn: "Zvikurukuriri zveBluetooth", NameNd: "Izikhulumi zeBluetooth"},
						{NameEn: "Home Theater Systems", NameSn: "Mabhaisikopo emumba", NameNd: "Ezokuzilibazisa"},
					},
				},
				{
					NameEn: "Cameras & Drone Technology",
					NameSn: "Zvekutora Mifananidzo",
					NameNd: "Izinto zokuthatha imifanekiso",
					Subcategories: []CategorySeed{
						{NameEn: "DSLR & Mirrorless", NameSn: "Makamera", NameNd: "Amakhamera"},
						{NameEn: "Action Cameras", NameSn: "Makamera emitambo", NameNd: "Amakhamera weAction"},
						{NameEn: "Drones", NameSn: "Dhironi", NameNd: "Amadrone"},
					},
				},
			},
		},
		{
			NameEn: "Bags, Watches & Jewelry",
			NameSn: "Mabhegi, Zvishongo neWachi",
			NameNd: "Izikhwama, Izihlobiso neMawashi",
			Subcategories: []CategorySeed{
				{
					NameEn: "Bags & Luggage",
					NameSn: "Mabhegi",
					NameNd: "Izikhwama",
					Subcategories: []CategorySeed{
						{NameEn: "Handbags & Totes", NameSn: "Mabhegi emuruoko", NameNd: "Izikhwama"},
						{NameEn: "Backpacks", NameSn: "Mabhegi emumusana", NameNd: "Izikhwama zommusana"},
						{NameEn: "Suitcases & Travel Bags", NameSn: "Mabhegi ekufamba", NameNd: "Imithwalo"},
					},
				},
				{
					NameEn: "Watches",
					NameSn: "Wachi",
					NameNd: "Amawashi",
					Subcategories: []CategorySeed{
						{NameEn: "Smartwatches", NameSn: "Smartwatches", NameNd: "Amawashi eTeknoloji"},
						{NameEn: "Mechanical & Quartz", NameSn: "Wachi dzechinyakare", NameNd: "Amawashi"},
					},
				},
				{
					NameEn: "Jewelry",
					NameSn: "Zvishongo",
					NameNd: "Izihlobiso",
					Subcategories: []CategorySeed{
						{NameEn: "Rings & Bands", NameSn: "Mhete", NameNd: "Izindandatho"},
						{NameEn: "Necklaces & Chains", NameSn: "Zvemuhuro", NameNd: "Imigexo"},
						{NameEn: "Earrings", NameSn: "Mhete dzenzeve", NameNd: "Izicaza"},
					},
				},
			},
		},
		{
			NameEn: "Home & Living",
			NameSn: "Musha neKugara",
			NameNd: "Ikhaya neKuhlala",
			Subcategories: []CategorySeed{
				{
					NameEn: "Furniture",
					NameSn: "Midziyo yemumba",
					NameNd: "Impahla zomuzi",
					Schema: furnitureSchema,
					Subcategories: []CategorySeed{
						{NameEn: "Sofas & Living Room", NameSn: "Masofa", NameNd: "Amasofa"},
						{NameEn: "Beds & Bedroom", NameSn: "Mibhedha", NameNd: "Imibheda"},
						{NameEn: "Desks & Office", NameSn: "Matafura", NameNd: "Amatafula"},
					},
				},
				{
					NameEn: "Kitchen & Dining",
					NameSn: "Zvemukitchen",
					NameNd: "Izinto zekhishi",
					Subcategories: []CategorySeed{
						{NameEn: "Cookware & Pots", NameSn: "Mapoto", NameNd: "Amapoto"},
						{NameEn: "Tableware & Plates", NameSn: "Mbiya neMagirazi", NameNd: "Izitsha"},
						{NameEn: "Kitchen Appliances", NameSn: "Zvishandiso zvemukitchen", NameNd: "Imishini yekhishi"},
					},
				},
				{
					NameEn: "Home Decor & Lighting",
					NameSn: "Zvekushongedza mumba",
					NameNd: "Zokuhlobisa indlu",
					Subcategories: []CategorySeed{
						{NameEn: "Rugs & Carpets", NameSn: "Makapeti", NameNd: "Amacansi"},
						{NameEn: "Curtains & Blinds", NameSn: "Machira emahwindo", NameNd: "Izilenge"},
						{NameEn: "Light Fixtures & Lamps", NameSn: "Mwenje", NameNd: "Izibane"},
					},
				},
			},
		},
		{
			NameEn: "Beauty & Personal Care",
			NameSn: "Runako neKuzvichengetedza",
			NameNd: "Ubuhle neZokuzikhathalela",
			Subcategories: []CategorySeed{
				{
					NameEn: "Skincare",
					NameSn: "Zvekuchengetedza Ganda",
					NameNd: "Zokukhathalela isikhumba",
					Schema: beautySchema,
					Subcategories: []CategorySeed{
						{NameEn: "Face Moisturizers", NameSn: "Mafuta eKumeso", NameNd: "Amafutha obuso"},
						{NameEn: "Sunscreens", NameSn: "Mafuta eZuva", NameNd: "Amafutha elanga"},
						{NameEn: "Cleansers & Face Wash", NameSn: "Sopo yeKumeso", NameNd: "Insipho yobuso"},
					},
				},
				{
					NameEn: "Makeup & Cosmetics",
					NameSn: "Zvekuzvizora",
					NameNd: "Zokuzichaza",
					Subcategories: []CategorySeed{
						{NameEn: "Foundations & Powders", NameSn: "Makeup dzechiso", NameNd: "Izinto zobuso"},
						{NameEn: "Lipsticks & Lip Care", NameSn: "Pende yemuromo", NameNd: "Izinto zomlomo"},
						{NameEn: "Eye Makeup", NameSn: "Zvemaziso", NameNd: "Izinto zamehlo"},
					},
				},
				{
					NameEn: "Hair Care",
					NameSn: "Zvevhudzi",
					NameNd: "Zokukhathalela inwele",
					Subcategories: []CategorySeed{
						{NameEn: "Shampoos & Conditioners", NameSn: "Shampoos", NameNd: "Ushampu"},
						{NameEn: "Hair Styling & Dyes", NameSn: "Pende yebvudzi", NameNd: "Izinto zenwele"},
						{NameEn: "Hair Dryers & Tools", NameSn: "Muchina webvudzi", NameNd: "Imishini yenwele"},
					},
				},
			},
		},
		{
			NameEn: "Groceries & Fresh Food",
			NameSn: "Chikafu neZvekudya",
			NameNd: "Ukudla neZokudla",
			Subcategories: []CategorySeed{
				{
					NameEn: "Pantry & Staples",
					NameSn: "Chikafu chemumba",
					NameNd: "Ukudla",
					Schema: foodSchema,
					Subcategories: []CategorySeed{
						{NameEn: "Rice & Grains", NameSn: "Mupunga neHupfu", NameNd: "Ilayisi lempuphu"},
						{NameEn: "Cooking Oils", NameSn: "Mafuta ekubikisa", NameNd: "Amafutha okupheka"},
						{NameEn: "Canned Goods", NameSn: "Chikafu chemumagaba", NameNd: "Ukudla okusemakotini"},
					},
				},
				{
					NameEn: "Snacks & Beverages",
					NameSn: "Zvekunanzva neZvinwiwa",
					NameNd: "Izinto zokudla-dla lezinwayo",
					Subcategories: []CategorySeed{
						{NameEn: "Chips & Biscuits", NameSn: "Mabhisikiti", NameNd: "Amabhisikiti"},
						{NameEn: "Coffee & Tea", NameSn: "Kofi neTii", NameNd: "Ikhefi leTiye"},
						{NameEn: "Soft Drinks & Juices", NameSn: "Zvinwiwa", NameNd: "Iziphuzo"},
					},
				},
			},
		},
		{
			NameEn: "Sports & Fitness",
			NameSn: "Mitambo neKugwinya",
			NameNd: "Imidlalo lezokuzivikela",
			Subcategories: []CategorySeed{
				{
					NameEn: "Fitness Gear",
					NameSn: "Zvekugwinyisa muviri",
					NameNd: "Izinto zokuzivikela",
					Subcategories: []CategorySeed{
						{NameEn: "Treadmills & Cardio", NameSn: "Michina yekumhanya", NameNd: "Imishini yokugijima"},
						{NameEn: "Dumbbells & Weights", NameSn: "Zvinorema", NameNd: "Izinsimbi"},
						{NameEn: "Yoga Mats", NameSn: "Yoga Mats", NameNd: "Amacansi weYoga"},
					},
				},
				{
					NameEn: "Outdoor Recreation",
					NameSn: "Zvekunze",
					NameNd: "Izangaphandle",
					Subcategories: []CategorySeed{
						{NameEn: "Tents & Camping", NameSn: "Matende", NameNd: "Amatende"},
						{NameEn: "Bicycles & Cycling", NameSn: "Mabhasikoro", NameNd: "Amabhasikili"},
						{NameEn: "Hiking & Climbing", NameSn: "Zvekukwira makomo", NameNd: "Ezokugwira"},
					},
				},
			},
		},
	}

	for _, parent := range categories {
		err := insertCategoryTree(ctx, pool, nil, parent, 1)
		if err != nil {
			fmt.Printf("Failed to seed category %s: %v\n", parent.NameEn, err)
			os.Exit(1)
		}
	}

	fmt.Println("Database seeded successfully with all 15 categories, subcategories, and translations!")
}

func insertCategoryTree(ctx context.Context, pool *pgxpool.Pool, parentID *uuid.UUID, seed CategorySeed, level int32) error {
	catSlug := slug.Make(seed.NameEn)
	if level > 1 {
		// Ensure unique subcategory slug
		catSlug = fmt.Sprintf("%s-%s", catSlug, uuid.New().String()[:4])
	}

	var schemaBytes []byte
	if seed.Schema != nil {
		schemaBytes, _ = json.Marshal(seed.Schema)
	}

	// Insert category
	var categoryID uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO categories (parent_id, slug, level, attribute_schema)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, parentID, catSlug, level, schemaBytes).Scan(&categoryID)
	if err != nil {
		return err
	}

	// Insert translations
	translations := []struct {
		lang string
		name string
	}{
		{"en", seed.NameEn},
		{"sn", func() string {
			if seed.NameSn != "" {
				return seed.NameSn
			}
			return seed.NameEn
		}()},
		{"nd", func() string {
			if seed.NameNd != "" {
				return seed.NameNd
			}
			return seed.NameEn
		}()},
	}

	for _, t := range translations {
		_, err := pool.Exec(ctx, `
			INSERT INTO category_translations (category_id, language, name)
			VALUES ($1, $2, $3)
		`, categoryID, t.lang, t.name)
		if err != nil {
			return err
		}
	}

	// Insert subcategories
	for _, sub := range seed.Subcategories {
		err := insertCategoryTree(ctx, pool, &categoryID, sub, level+1)
		if err != nil {
			return err
		}
	}

	return nil
}
