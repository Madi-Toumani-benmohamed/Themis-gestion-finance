package api

import "log"

// defaultRules : règles d'auto-tri de départ, associées par NOM de catégorie
// (résolu vers l'id propre à chaque utilisateur au moment du seed).
var defaultRules = []struct {
	Pattern  string
	Category string
}{
	{"CARREFOUR", "Alimentation"}, {"LECLERC", "Alimentation"}, {"AUCHAN", "Alimentation"},
	{"LIDL", "Alimentation"}, {"INTERMARCHE", "Alimentation"}, {"MONOPRIX", "Alimentation"},
	{"FRANPRIX", "Alimentation"}, {"CASINO", "Alimentation"},
	{"LOYER", "Logement"}, {"EDF", "Logement"}, {"ENGIE", "Logement"}, {"VEOLIA", "Logement"},
	{"SNCF", "Transport"}, {"UBER", "Transport"}, {"TOTALENERGIES", "Transport"},
	{"RATP", "Transport"}, {"ESSENCE", "Transport"},
	{"NETFLIX", "Divertissement"}, {"SPOTIFY", "Divertissement"}, {"DISNEY", "Divertissement"},
	{"PHARMACIE", "Santé et Bien-être"}, {"DOCTEUR", "Santé et Bien-être"},
	{"FRAIS BANCAIRES", "Autres"}, {"COTISATION", "Autres"},
}

// defaultCategories : les catégories offertes à chaque nouvel utilisateur, pour
// qu'il démarre avec une structure prête à l'emploi plutôt qu'une app vide.
var defaultCategories = []struct {
	Name  string
	Color string
}{
	{"Logement", "#4C6FBF"},
	{"Transport", "#3F8F7B"},
	{"Alimentation", "#C77D2E"},
	{"Santé et Bien-être", "#D6455D"},
	{"Divertissement", "#8A5CD1"},
	{"Épargne et Investissements", "#2E8B57"},
	{"Vie Professionnelle", "#3D6B99"},
	{"Famille", "#B5548A"},
	{"Autres", "#8A8A99"},
}

// La catégorie de transfert interne est créée à part (elle porte le drapeau
// is_transfer) : virements entre comptes personnels, neutres dans les stats.
const transferCategoryName = "Transfert interne"

// seedUserData installe les données de départ d'un nouvel utilisateur :
// un compte principal et les catégories par défaut. Idempotent.
func (s *Server) seedUserData(userID int64) {
	// Compte principal. Si un compte orphelin existe (résidu d'installation
	// mono-utilisateur), le premier utilisateur créé l'adopte ; sinon on crée
	// un compte neuf. Les utilisateurs suivants ont toujours leur propre compte.
	res, err := s.DB.Exec(`UPDATE accounts SET owner_id = $1 WHERE owner_id IS NULL`, userID)
	adopted := int64(0)
	if err == nil {
		adopted, _ = res.RowsAffected()
	}
	if adopted == 0 {
		_, err = s.DB.Exec(`
			INSERT INTO accounts (name, kind, owner_id)
			SELECT 'Compte courant', 'courant', $1
			WHERE NOT EXISTS (SELECT 1 FROM accounts WHERE owner_id = $1)`, userID)
		if err != nil {
			log.Printf("seedUserData: compte pour user %d: %v", userID, err)
		}
	}

	for _, c := range defaultCategories {
		_, err := s.DB.Exec(`
			INSERT INTO domains (name, parent_id, color, owner_id)
			VALUES ($1, NULL, $2, $3)
			ON CONFLICT (owner_id, name) WHERE parent_id IS NULL DO NOTHING`,
			c.Name, c.Color, userID)
		if err != nil {
			log.Printf("seedUserData: catégorie %q pour user %d: %v", c.Name, userID, err)
		}
	}

	// Catégorie de transfert interne (marquée is_transfer, exclue des stats).
	if _, err := s.DB.Exec(`
		INSERT INTO domains (name, parent_id, color, owner_id, is_transfer)
		VALUES ($1, NULL, $2, $3, TRUE)
		ON CONFLICT (owner_id, name) WHERE parent_id IS NULL DO NOTHING`,
		transferCategoryName, "#7A8290", userID); err != nil {
		log.Printf("seedUserData: catégorie transfert pour user %d: %v", userID, err)
	}

	// Règles d'auto-tri de départ, résolues vers les catégories de l'utilisateur.
	for _, ru := range defaultRules {
		_, err := s.DB.Exec(`
			INSERT INTO rules (pattern, domain_id, owner_id)
			SELECT $1, d.id, $3 FROM domains d
			WHERE d.owner_id = $3 AND d.parent_id IS NULL AND d.name = $2
			ON CONFLICT (owner_id, lower(pattern)) DO NOTHING`,
			ru.Pattern, ru.Category, userID)
		if err != nil {
			log.Printf("seedUserData: règle %q pour user %d: %v", ru.Pattern, userID, err)
		}
	}
}
