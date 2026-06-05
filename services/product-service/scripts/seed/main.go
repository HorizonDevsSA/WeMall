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
				{NameEn: "Women's Clothing", NameSn: "Zvipfeko zveMadzimai", NameNd: "Izokugqoka zeSifazane", Schema: clothingSchema},
				{NameEn: "Men's Clothing", NameSn: "Zvipfeko zveVarume", NameNd: "Izokugqoka zeSilisa", Schema: clothingSchema},
				{NameEn: "Children's Clothing", NameSn: "Zvipfeko zveVana", NameNd: "Izokugqoka zeBantwana", Schema: clothingSchema},
				{NameEn: "Underwear", NameSn: "Zvemukati", NameNd: "Izomgogodla"},
				{NameEn: "Shoes & Boots", NameSn: "Shangu neBhuts", NameNd: "Izicathulo", Schema: shoesSchema},
				{NameEn: "Streetwear", NameSn: "ZveSitarata", NameNd: "Izomgwaqo"},
				{NameEn: "Traditional Clothing", NameSn: "Zvipfeko zveChinyakare", NameNd: "Izokugqoka zeSintu"},
				{NameEn: "Fashion Accessories", NameSn: "Zvishongo zveFashoni", NameNd: "Izihlobiso zeFashoni"},
			},
		},
		{
			NameEn: "Bags, Jewelry & Watches",
			NameSn: "Mabhegi, Zvishongo neWachi",
			NameNd: "Izikhwama, Izihlobiso neMawashi",
			Subcategories: []CategorySeed{
				{NameEn: "Handbags", NameSn: "Mabhegi emuRuoko", NameNd: "Izikhwama zomDwadla"},
				{NameEn: "Backpacks", NameSn: "Mabhegi emuMusana", NameNd: "Izikhwama zomMusana"},
				{NameEn: "Wallets", NameSn: "Zvikwama zveMari", NameNd: "Izikhwama zeMali"},
				{NameEn: "Jewelry", NameSn: "Zvishongo", NameNd: "Izihlobiso"},
				{NameEn: "Smartwatches", NameSn: "Wachi dzeMagetsi", NameNd: "Mawashi eTeknoloji"},
				{NameEn: "Fashion Watches", NameSn: "Wachi dzeFashoni", NameNd: "Mawashi eFashoni"},
			},
		},
		{
			NameEn: "Electronics & Technology",
			NameSn: "ZveMagetsi neTekinoroji",
			NameNd: "Ezomanyalo neTeknoloji",
			Subcategories: []CategorySeed{
				{NameEn: "Smartphones", NameSn: "Nharembozha", NameNd: "Ucingo lweSandla", Schema: phoneSchema},
				{NameEn: "Phone Accessories", NameSn: "ZveNharembozha", NameNd: "Izinto zefoni"},
				{NameEn: "Chargers & Cables", NameSn: "ZveKuchajisa", NameNd: "Izinto zokuchajisa"},
				{NameEn: "Earbuds & Headphones", NameSn: "ZvekuTeereresa", NameNd: "Izinto zokulalela"},
				{NameEn: "Computer Accessories", NameSn: "ZveMakombiyuta", NameNd: "Izinto zeKhompyutha"},
				{NameEn: "Gaming Accessories", NameSn: "ZveMitambo yeKombiyuta", NameNd: "Izinto zemidlalo"},
				{NameEn: "Smart Home Devices", NameSn: "Zvemumba zveMagetsi", NameNd: "Izinto zomuzi zeTeknoloji"},
				{NameEn: "Cameras & Photography", NameSn: "Zvekutora Mifananidzo", NameNd: "Izinto zokuthatha imifanekiso"},
			},
		},
		{
			NameEn: "Beauty & Personal Care",
			NameSn: "Runako neKuzvichengetedza",
			NameNd: "Ubuhle neZokuzikhathalela",
			Subcategories: []CategorySeed{
				{NameEn: "Skincare", NameSn: "Zvekuchengetedza Ganda", NameNd: "Zokukhathalela isikhumba", Schema: beautySchema},
				{NameEn: "Makeup", NameSn: "Zvekuzvizora", NameNd: "Zokuzichaza"},
				{NameEn: "Hair Care", NameSn: "Zvevhudzi", NameNd: "Zokukhathalela inwele"},
				{NameEn: "Perfumes", NameSn: "Zvinonhuhwirira", NameNd: "Izamakha"},
				{NameEn: "Beauty Tools", NameSn: "Zvishandiso zverunako", NameNd: "Izinto zokuzichaza"},
			},
		},
		{
			NameEn: "Home & Living",
			NameSn: "Musha neKugara",
			NameNd: "Ikhaya neKuhlala",
			Subcategories: []CategorySeed{
				{NameEn: "Furniture", NameSn: "Midziyo yemumba", NameNd: "Impahla zomuzi", Schema: furnitureSchema},
				{NameEn: "Home Décor", NameSn: "Zvekushongedza mumba", NameNd: "Zokuhlobisa indlu"},
				{NameEn: "Lighting", NameSn: "Mwenje", NameNd: "Izibane"},
				{NameEn: "Bedding", NameSn: "Zvepamubhedha", NameNd: "Izinto zomubheda"},
				{NameEn: "Kitchenware", NameSn: "Zvemukitchen", NameNd: "Izinto zekhishi"},
				{NameEn: "Storage", NameSn: "Zvekuchengetera midziyo", NameNd: "Izinto zokuchengetela"},
			},
		},
		{
			NameEn: "Toys, Hobbies & Kids",
			NameSn: "Zvekutambisa, Zvekuvaraidza neVana",
			NameNd: "Amathoyizi, Ezokuzilibazisa neBantwana",
			Subcategories: []CategorySeed{
				{NameEn: "Baby Toys", NameSn: "Zvekutambisa zveVacheche", NameNd: "Amathoyizi eNgane"},
				{NameEn: "Action Figures", NameSn: "Zvidhori", NameNd: "Izithombe"},
				{NameEn: "RC Cars", NameSn: "Motokari dzeRemoti", NameNd: "Izimoto zeRemoti"},
				{NameEn: "Board Games", NameSn: "Mitambo yemabhodhi", NameNd: "Imidlalo yebhodi"},
				{NameEn: "Plush Toys", NameSn: "Zvidhori zvakapfava", NameNd: "Amathoyizi athambileyo"},
			},
		},
		{
			NameEn: "Sports & Outdoors",
			NameSn: "Mitambo neZvekunze",
			NameNd: "Imidlalo neZangaphandle",
			Subcategories: []CategorySeed{
				{NameEn: "Activewear", NameSn: "Zvekupfeka zvemitambo", NameNd: "Izokugqoka zemidlalo"},
				{NameEn: "Running Shoes", NameSn: "Shangu dzekumhanya", NameNd: "Izicathulo zokugijima"},
				{NameEn: "Camping Gear", NameSn: "Zvekumisasa", NameNd: "Izinto zokukampa"},
				{NameEn: "Fitness Equipment", NameSn: "Zvekugwinyisa muviri", NameNd: "Izinto zokuzivikela"},
				{NameEn: "Yoga Gear", NameSn: "ZveYoga", NameNd: "Izinto zeYoga"},
			},
		},
		{
			NameEn: "Car & Motorcycle Accessories",
			NameSn: "ZveMotokari neMidhudhudhu",
			NameNd: "Izinto zeMota neZithuthuthu",
			Subcategories: []CategorySeed{
				{NameEn: "Car Electronics", NameSn: "Zvemagetsi zvemotokari", NameNd: "Ezomanyalo zemota"},
				{NameEn: "Seat Covers", NameSn: "Zvekufukidza zvigaro", NameNd: "Izinto zokugubuzela izitulo"},
				{NameEn: "Cleaning Tools", NameSn: "Zvekuchenesa", NameNd: "Izinto zokuhlanza"},
				{NameEn: "Moto Gear", NameSn: "Zvemidhudhudhu", NameNd: "Izinto zezithuthuthu"},
			},
		},
		{
			NameEn: "Food & Groceries",
			NameSn: "Chikafu neZvekudya",
			NameNd: "Ukudla neZokudla",
			Subcategories: []CategorySeed{
				{NameEn: "Snacks", NameSn: "Zvekunanzva", NameNd: "Izinto zokudla-dla", Schema: foodSchema},
				{NameEn: "Tea & Coffee", NameSn: "Tii neKofi", NameNd: "Itiye leKhofi"},
				{NameEn: "Instant Meals", NameSn: "Chikafu chinokurumidza", NameNd: "Ukudla okusheshayo"},
				{NameEn: "Organic Food", NameSn: "Chikafu chechinyakare", NameNd: "Ukudla kwemvelo"},
			},
		},
		{
			NameEn: "Pet Supplies",
			NameSn: "Zvezvipfuyo",
			NameNd: "Izinto zezifuyo",
			Subcategories: []CategorySeed{
				{NameEn: "Dog Food", NameSn: "Chikafu cheImbwa", NameNd: "Ukudla kweZinja"},
				{NameEn: "Cat Food", NameSn: "Chikafu cheKatsi", NameNd: "Ukudla kweZikati"},
				{NameEn: "Pet Toys", NameSn: "Zvekutambisa zvezvipfuyo", NameNd: "Amathoyizi ezifuyo"},
				{NameEn: "Pet Beds", NameSn: "Mibhedha yezvipfuyo", NameNd: "Imibheda yezifuyo"},
			},
		},
		{
			NameEn: "Books, Stationery & Office",
			NameSn: "Mabhuku, Zvekunyora neHofisi",
			NameNd: "Amabhuku, Izinto zokubhala neOfisi",
			Subcategories: []CategorySeed{
				{NameEn: "Novels", NameSn: "Ngano", NameNd: "Izindaba"},
				{NameEn: "Notebooks", NameSn: "Mabhuku ekunyorera", NameNd: "Amabhuku okubhala"},
				{NameEn: "Pens & Pencils", NameSn: "Zvinyoreso", NameNd: "Izinto zokubhala"},
				{NameEn: "Art Supplies", NameSn: "Zvekudhirowesa", NameNd: "Izinto zokuzoba"},
			},
		},
		{
			NameEn: "Musical Instruments",
			NameSn: "Zviridzwa zvoMhanzi",
			NameNd: "Izinto zokudlala umculo",
			Subcategories: []CategorySeed{
				{NameEn: "Guitars", NameSn: "Gitare", NameNd: "Igitari"},
				{NameEn: "Pianos & Keyboards", NameSn: "Piyano", NameNd: "Ipiyano"},
				{NameEn: "Drums", NameSn: "Ngoma", NameNd: "Izighubu"},
			},
		},
		{
			NameEn: "Tools & Home Improvement",
			NameSn: "Zvishandiso neKuvandudza Musha",
			NameNd: "Izinto zokusebenza neKuvuselela Ikhaya",
			Subcategories: []CategorySeed{
				{NameEn: "Power Tools", NameSn: "Zvishandiso zvemagetsi", NameNd: "Izinto zokusebenza zeTeknoloji"},
				{NameEn: "Hand Tools", NameSn: "Zvishandiso zvemaoko", NameNd: "Izinto zokusebenza zezandla"},
				{NameEn: "Paint", NameSn: "Pende", NameNd: "Upende"},
			},
		},
		{
			NameEn: "Health & Wellness",
			NameSn: "Utano neKugwinya",
			NameNd: "Impilo neZokuzivikela",
			Subcategories: []CategorySeed{
				{NameEn: "Supplements", NameSn: "Mavitamini", NameNd: "Amavithamini"},
				{NameEn: "Medical Devices", NameSn: "Midziyo yekurapa", NameNd: "Izinto zokwelapha"},
				{NameEn: "Massagers", NameSn: "Zvekuvhura muviri", NameNd: "Izinto zokuvuselela umzimba"},
			},
		},
		{
			NameEn: "Arts, Crafts & Sewing",
			NameSn: "Unyanzvi, Zvemaoko neKusona",
			NameNd: "Ubuci, Izinto zezandla neZokuthunga",
			Subcategories: []CategorySeed{
				{NameEn: "Fabric & Yarn", NameSn: "Machira neshinda", NameNd: "Amalula lenyambo"},
				{NameEn: "Craft Kits", NameSn: "Zvishandiso zvezvemaoko", NameNd: "Izinto zezandla"},
				{NameEn: "Sewing Tools", NameSn: "Zvekusonesa", NameNd: "Izinto zokuthunga"},
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
