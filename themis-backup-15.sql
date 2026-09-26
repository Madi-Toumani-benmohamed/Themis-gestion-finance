--
-- PostgreSQL database dump
--

\restrict JkGxOZXZp4jBtqLWMScW9wdosqiab0oTLgO948iqL4WDZjOdVPoeoipRIhyGej8

-- Dumped from database version 16.14
-- Dumped by pg_dump version 16.14

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: account_snapshots; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.account_snapshots (
    id bigint NOT NULL,
    account_id bigint NOT NULL,
    snap_date date NOT NULL,
    balance numeric(12,2) NOT NULL
);


ALTER TABLE public.account_snapshots OWNER TO themis;

--
-- Name: account_snapshots_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.account_snapshots_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.account_snapshots_id_seq OWNER TO themis;

--
-- Name: account_snapshots_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.account_snapshots_id_seq OWNED BY public.account_snapshots.id;


--
-- Name: accounts; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.accounts (
    id bigint NOT NULL,
    name text NOT NULL,
    kind text DEFAULT 'courant'::text NOT NULL,
    interest_rate numeric(5,2),
    deposit_limit numeric(12,2),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    owner_id bigint
);


ALTER TABLE public.accounts OWNER TO themis;

--
-- Name: accounts_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.accounts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.accounts_id_seq OWNER TO themis;

--
-- Name: accounts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.accounts_id_seq OWNED BY public.accounts.id;


--
-- Name: analyses; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.analyses (
    id integer NOT NULL,
    month text NOT NULL,
    content text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    owner_id bigint
);


ALTER TABLE public.analyses OWNER TO themis;

--
-- Name: analyses_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.analyses_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.analyses_id_seq OWNER TO themis;

--
-- Name: analyses_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.analyses_id_seq OWNED BY public.analyses.id;


--
-- Name: budgets; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.budgets (
    id bigint NOT NULL,
    monthly_limit numeric(12,2) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    domain_id bigint,
    owner_id bigint,
    CONSTRAINT budgets_monthly_limit_check CHECK ((monthly_limit > (0)::numeric))
);


ALTER TABLE public.budgets OWNER TO themis;

--
-- Name: budgets_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.budgets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.budgets_id_seq OWNER TO themis;

--
-- Name: budgets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.budgets_id_seq OWNED BY public.budgets.id;


--
-- Name: credits; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.credits (
    id bigint NOT NULL,
    name text NOT NULL,
    monthly_amount numeric(12,2) NOT NULL,
    months_total integer NOT NULL,
    start_month character(7) NOT NULL,
    note text DEFAULT ''::text NOT NULL,
    owner_id bigint,
    CONSTRAINT credits_monthly_amount_check CHECK ((monthly_amount > (0)::numeric)),
    CONSTRAINT credits_months_total_check CHECK ((months_total > 0))
);


ALTER TABLE public.credits OWNER TO themis;

--
-- Name: credits_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.credits_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.credits_id_seq OWNER TO themis;

--
-- Name: credits_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.credits_id_seq OWNED BY public.credits.id;


--
-- Name: domains; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.domains (
    id bigint NOT NULL,
    name text NOT NULL,
    parent_id bigint,
    color text DEFAULT '#66736C'::text NOT NULL,
    owner_id bigint,
    is_transfer boolean DEFAULT false NOT NULL
);


ALTER TABLE public.domains OWNER TO themis;

--
-- Name: domains_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.domains_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.domains_id_seq OWNER TO themis;

--
-- Name: domains_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.domains_id_seq OWNED BY public.domains.id;


--
-- Name: merchant_aliases; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.merchant_aliases (
    id bigint NOT NULL,
    owner_id bigint NOT NULL,
    match_key text NOT NULL,
    display text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    is_pattern boolean DEFAULT false NOT NULL
);


ALTER TABLE public.merchant_aliases OWNER TO themis;

--
-- Name: merchant_aliases_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.merchant_aliases_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.merchant_aliases_id_seq OWNER TO themis;

--
-- Name: merchant_aliases_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.merchant_aliases_id_seq OWNED BY public.merchant_aliases.id;


--
-- Name: rules; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.rules (
    id integer NOT NULL,
    pattern text NOT NULL,
    domain_id bigint NOT NULL,
    owner_id bigint,
    tags text DEFAULT ''::text NOT NULL
);


ALTER TABLE public.rules OWNER TO themis;

--
-- Name: rules_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.rules_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.rules_id_seq OWNER TO themis;

--
-- Name: rules_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.rules_id_seq OWNED BY public.rules.id;


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.schema_migrations (
    name text NOT NULL,
    applied_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO themis;

--
-- Name: sessions; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.sessions (
    token text NOT NULL,
    user_id bigint NOT NULL,
    expires_at timestamp with time zone NOT NULL
);


ALTER TABLE public.sessions OWNER TO themis;

--
-- Name: settings; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.settings (
    key text NOT NULL,
    value jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    owner_id bigint
);


ALTER TABLE public.settings OWNER TO themis;

--
-- Name: statements; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.statements (
    id integer NOT NULL,
    filename text NOT NULL,
    period_start date,
    period_end date,
    opening_balance numeric(12,2),
    closing_balance numeric(12,2),
    parser text DEFAULT 'local'::text NOT NULL,
    validated boolean DEFAULT false NOT NULL,
    uploaded_at timestamp with time zone DEFAULT now() NOT NULL,
    account_id bigint DEFAULT 1 NOT NULL,
    owner_id bigint
);


ALTER TABLE public.statements OWNER TO themis;

--
-- Name: statements_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.statements_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.statements_id_seq OWNER TO themis;

--
-- Name: statements_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.statements_id_seq OWNED BY public.statements.id;


--
-- Name: tag_folder_prefs; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.tag_folder_prefs (
    owner_id bigint NOT NULL,
    tag_name text NOT NULL,
    state text NOT NULL,
    CONSTRAINT tag_folder_prefs_state_check CHECK ((state = ANY (ARRAY['pinned'::text, 'hidden'::text])))
);


ALTER TABLE public.tag_folder_prefs OWNER TO themis;

--
-- Name: tags; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.tags (
    id bigint NOT NULL,
    name text NOT NULL,
    owner_id bigint
);


ALTER TABLE public.tags OWNER TO themis;

--
-- Name: tags_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.tags_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.tags_id_seq OWNER TO themis;

--
-- Name: tags_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.tags_id_seq OWNED BY public.tags.id;


--
-- Name: taxes; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.taxes (
    id bigint NOT NULL,
    name text NOT NULL,
    amount numeric(12,2) NOT NULL,
    due_date date NOT NULL,
    paid boolean DEFAULT false NOT NULL,
    owner_id bigint,
    CONSTRAINT taxes_amount_check CHECK ((amount > (0)::numeric))
);


ALTER TABLE public.taxes OWNER TO themis;

--
-- Name: taxes_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.taxes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.taxes_id_seq OWNER TO themis;

--
-- Name: taxes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.taxes_id_seq OWNED BY public.taxes.id;


--
-- Name: transaction_tags; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.transaction_tags (
    transaction_id bigint NOT NULL,
    tag_id bigint NOT NULL
);


ALTER TABLE public.transaction_tags OWNER TO themis;

--
-- Name: transactions; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.transactions (
    id integer NOT NULL,
    statement_id integer,
    tx_date date NOT NULL,
    label text NOT NULL,
    amount numeric(12,2) NOT NULL,
    hash text NOT NULL,
    domain_id bigint,
    owner_id bigint,
    display_label text DEFAULT ''::text NOT NULL,
    merchant text DEFAULT ''::text NOT NULL,
    label_kind text DEFAULT ''::text NOT NULL
);


ALTER TABLE public.transactions OWNER TO themis;

--
-- Name: transactions_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.transactions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.transactions_id_seq OWNER TO themis;

--
-- Name: transactions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.transactions_id_seq OWNED BY public.transactions.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.users (
    id bigint NOT NULL,
    username text NOT NULL,
    password_hash text NOT NULL,
    role text DEFAULT 'user'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT users_role_check CHECK ((role = ANY (ARRAY['admin'::text, 'user'::text])))
);


ALTER TABLE public.users OWNER TO themis;

--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.users_id_seq OWNER TO themis;

--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: account_snapshots id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.account_snapshots ALTER COLUMN id SET DEFAULT nextval('public.account_snapshots_id_seq'::regclass);


--
-- Name: accounts id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.accounts ALTER COLUMN id SET DEFAULT nextval('public.accounts_id_seq'::regclass);


--
-- Name: analyses id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.analyses ALTER COLUMN id SET DEFAULT nextval('public.analyses_id_seq'::regclass);


--
-- Name: budgets id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.budgets ALTER COLUMN id SET DEFAULT nextval('public.budgets_id_seq'::regclass);


--
-- Name: credits id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.credits ALTER COLUMN id SET DEFAULT nextval('public.credits_id_seq'::regclass);


--
-- Name: domains id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.domains ALTER COLUMN id SET DEFAULT nextval('public.domains_id_seq'::regclass);


--
-- Name: merchant_aliases id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.merchant_aliases ALTER COLUMN id SET DEFAULT nextval('public.merchant_aliases_id_seq'::regclass);


--
-- Name: rules id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.rules ALTER COLUMN id SET DEFAULT nextval('public.rules_id_seq'::regclass);


--
-- Name: statements id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.statements ALTER COLUMN id SET DEFAULT nextval('public.statements_id_seq'::regclass);


--
-- Name: tags id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.tags ALTER COLUMN id SET DEFAULT nextval('public.tags_id_seq'::regclass);


--
-- Name: taxes id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.taxes ALTER COLUMN id SET DEFAULT nextval('public.taxes_id_seq'::regclass);


--
-- Name: transactions id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.transactions ALTER COLUMN id SET DEFAULT nextval('public.transactions_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Data for Name: account_snapshots; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.account_snapshots (id, account_id, snap_date, balance) FROM stdin;
2	2	2026-07-18	233.00
5	5	2026-07-18	1200.00
6	1	2026-07-20	58.00
7	7	2026-07-20	2962.00
8	8	2026-07-20	770.00
9	1	2026-09-06	978.00
10	2	2026-09-06	133.00
11	5	2026-09-06	1900.00
12	7	2026-09-06	2829.29
13	8	2026-09-06	841.00
\.


--
-- Data for Name: accounts; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.accounts (id, name, kind, interest_rate, deposit_limit, created_at, owner_id) FROM stdin;
1	Compte courant	courant	\N	\N	2026-07-18 13:33:35.382182+00	1
2	Livret A	livret_a	\N	\N	2026-07-18 16:00:48.667578+00	1
5	Cryptomonaie	autre	\N	\N	2026-07-18 16:01:18.552838+00	1
6	Compte courant	courant	\N	\N	2026-07-18 17:42:32.763227+00	2
7	PEL	pel	\N	\N	2026-07-20 11:45:03.447029+00	1
8	PEA	bourse	\N	\N	2026-07-20 11:45:28.042904+00	1
\.


--
-- Data for Name: analyses; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.analyses (id, month, content, created_at, owner_id) FROM stdin;
1	2026-02	### Vue d'ensemble\n\nLe mois de février 2026 a été déficitaire avec un solde négatif de -121,02 €, ce qui indique une dépense légèrement supérieure aux revenus. La catégorie "Famille" représente plus de la moitié des dépenses totales.\n\n### Points d'attention\n\n- **Famille (3022,09 €)** : Cette dépense est très élevée par rapport au mois précédent (2170,00 €) et aux mois précédents. Il faut vérifier les détails pour comprendre la cause.\n- **Vie Professionnelle (3045,00 €)** : Ce poste a connu une forte augmentation depuis le mois de janvier 2026 (143,10 €). Il est important d'identifier l'origine de cette dépense.\n\n### Actions concrètes\n\n- **Réduire les frais "Famille"** : Si la dépense élevée en février est due à un événement unique comme une fête ou des cadeaux, envisagez de limiter ces dépenses futures. Réduction possible : 500 €.\n- **Analyser les frais professionnels** : Identifiez l'origine du montant élevé pour la "Vie Professionnelle". Si c'est un paiement unique, ajustez votre budget en conséquence. Réduction possible : 2901,90 € (pour revenir au niveau de janvier).\n- **Optimiser les transferts internes** : Le transfert interne de 5000 € est élevé et peut être réduit si non nécessaire. Réduction possible : 2500 €.\n\n### Tendance\n\nComparativement aux mois précédents, la dépense "Famille" a considérablement augmenté en février (3022,09 € contre 660 € en décembre et 2170 € en janvier). De même, les frais de "Vie Professionnelle" ont fortement grimpé à 3045 € depuis le niveau bas de janvier (143,10 €).	2026-07-18 15:21:39.506497+00	1
6	2026-05	### Vue d'ensemble\n\nLe mois de mai a été marqué par une forte baisse des dépenses dans plusieurs catégories, mais le solde négatif persiste avec un déficit de 151,41 €. La réduction significative des coûts liés à la famille et au logement est notable.\n\n### Points d'attention\n\n- **Famille** : Une baisse drastique de 95% (88,68 € vs 1748 € en moyenne) suggère un changement majeur dans les dépenses familiales.\n- **Logement** : La diminution de 91% (97,34 € vs 1047 € en moyenne) indique une modification importante des frais liés au logement.\n- **Investissements et Autres** : Les baisses respectives de 58% (22 € vs 53 €) pour les investissements et de 96% (6 € vs 148 €) pour d'autres dépenses signalent des réductions importantes dans ces domaines.\n\n### Actions concrètes\n\n- **Réévaluer la situation familiale** : Si la baisse des dépenses familiales est due à un changement de statut familial (divorce, séparation), il faut ajuster le budget en conséquence. Par exemple, réduire les frais liés aux enfants ou au conjoint.\n- **Analyser les coûts du logement** : Étudier pourquoi les dépenses liées au logement ont diminué de manière significative et s'assurer qu'il n'y a pas d'impayés cachés. Par exemple, vérifier si le loyer est toujours à jour.\n- **Revoir l'allocation budgétaire** : Augmenter la part allouée aux dépenses alimentaires pour compenser les baisses dans d'autres catégories et maintenir un équilibre financier stable.\n\n### Tendance\n\nComparé aux mois précédents, mai a connu une réduction massive des dépenses familiales et de logement. Les autres catégories comme l'investissement et le transport ont également diminué, mais moins radicalement. La catégorie Santé et Bien-être n'est pas présente ce mois-ci, contrairement à la moyenne précédente.	2026-07-31 11:58:46.240836+00	1
2	2026-06	### Vue d'ensemble\n\nCe mois-ci, le budget est bien équilibré avec un solde positif de 1551.13 €. Cependant, plusieurs dépenses importantes ont fortement augmenté, nécessitant une attention particulière.\n\n### Points d'attention\n\n- **Famille** : La dépense a grimpé à 2925.09 € ce mois-ci, soit une hausse de 270% par rapport aux mois précédents.\n- **Divertissement** : Les frais ont bondi à 978.98 €, en hausse de 445%.\n- **Amis et Transport** : Ces deux catégories ont également connu une augmentation significative avec respectivement +111% pour les amis (95 €) et +71% pour le transport (62 €).\n\n### Actions concrètes\n\n1. **Réévaluer la dépense "Famille"** : Identifiez les causes de cette forte hausse (par exemple, des frais médicaux non habituels ou des achats importants). Si nécessaire, ajustez votre budget pour inclure ces coûts futurs.\n2. **Contrôler le divertissement** : Réduisez vos dépenses en divertissement à un niveau plus réaliste et budgétisé (par exemple, 300 € au lieu de 978.98 €). Cela pourrait inclure des activités gratuites ou moins coûteuses.\n3. **Économiser sur les transports** : Utilisez davantage le covoiturage ou les transports en commun pour réduire vos frais à environ 40 € par mois.\n\n### Tendance\n\n- La catégorie "Famille" a connu une hausse spectaculaire de 270%.\n- Le divertissement est également en forte hausse (+445%), reflétant peut-être des dépenses imprévues ou excessives.\n- Les frais d'Alimentation et de Logement ont baissé respectivement de 34% et 91%, ce qui pourrait indiquer une réduction des achats alimentaires non essentiels ou un changement dans les coûts liés au logement (par exemple, paiements anticipés).\n- Les dépenses en Amis et Transport ont augmenté (+111% et +71%), tandis que l'Épargne et Investissements restent stables.	2026-07-31 11:57:14.178427+00	1
7	2026-04	### Vue d'ensemble\n\nLe mois de 2026-04 a été marqué par une forte augmentation des dépenses liées au logement et aux autres frais, compensée par une baisse significative dans les catégories famille et vie professionnelle. Le solde négatif suggère un besoin d'ajustements pour améliorer la gestion financière globale.\n\n### Points d'attention\n\n- **Logement** : 2947 €, soit une hausse de 181% par rapport aux mois précédents.\n- **Autres frais** : 238 €, en hausse de 240%, probablement liés à des dépenses imprévues ou non budgétisées.\n- **Divertissement** : 331 €, une augmentation de 85% par rapport aux mois précédents.\n\n### Actions concrètes\n\n1. **Réévaluer les frais de logement** : Identifier et négocier éventuellement les coûts liés au logement (loyer, charges) pour réduire cette dépense majeure.\n2. **Établir un budget pour autres frais** : Créer une ligne budgétaire spécifique pour les dépenses imprévues afin de limiter leur impact sur le solde global.\n3. **Réduire les frais de divertissement** : Limiter les sorties et activités coûteuses en optant pour des alternatives moins onéreuses.\n\n### Tendance\n\n- Les dépenses liées au logement ont considérablement augmenté ce mois-ci, passant d'une moyenne de 1048 € à 2947 €.\n- La catégorie famille a connu une baisse significative (de 2236 € en moyenne à 391 €).\n- Les frais de divertissement ont également augmenté, passant de 179 € à 331 €.\n- Le transport est passé d'une moyenne de 193 € à seulement 6 € ce mois-ci.	2026-07-31 12:00:23.464425+00	1
16	2026-03	### Vue d'ensemble\n\nLe mois de mars est marqué par une forte dépendance aux dépenses familiales, malgré des baisses significatives dans plusieurs autres catégories. Le solde négatif suggère un besoin urgent d'équilibrer les revenus et les dépenses.\n\n### Points d'attention\n\n- **Famille** : 1894 € (EN BAISSE de 17%) — Bien que la dépense soit en baisse, elle reste très élevée.\n- **Alimentation** : 278 € (EN HAUSSE de 32%) — Une augmentation importante qui doit être surveillée.\n- **Logement** : 97 € (EN BAISSE de 91%) — Cette forte baisse est inhabituelle et mérite une vérification.\n\n### Actions concrètes\n\n1. **Réduire les dépenses alimentaires** : Réorganisez vos achats pour limiter la dépense à 200 € par mois.\n2. **Vérifier le coût du logement** : Contactez votre bailleur ou prestataire de services pour comprendre cette baisse soudaine et prévenir d'éventuels problèmes futurs.\n3. **Économiser sur les transports** : Utilisez plus souvent des alternatives gratuites comme la marche ou le vélo, limitant ainsi vos dépenses à 50 € par mois.\n\n### Tendance\n\n- La catégorie "Famille" a connu une baisse significative mais reste élevée.\n- L'alimentation et les autres dépenses ont augmenté de manière notable.\n- Le logement a enregistré une forte diminution, ce qui est inhabituel.\n- Les dépenses pour le divertissement et l'interaction sociale (Amis) sont absentes ce mois-ci.	2026-07-31 12:01:20.678697+00	1
9	2026-07	### Vue d'ensemble\nLe mois est fortement déficitaire avec un solde de -1465,53 €, car vos revenus ne couvrent pas les charges fixes et variables. Cette situation est accentuée par la structure de votre activité d'indépendant et le rythme des paiements du loyer.\n\n### Points d'attention\n- **Télécommunications** : EN HAUSSE de 73% (24 € ce mois contre 14 € habituellement).\n- **Famille** : Poste important à 833,00 €, bien qu'en baisse de 27%.\n\n### Actions concrètes\n1. **Provisionner le loyer** : Étant donné que vous payez trimestriellement, mettez systématiquement 350 € de côté chaque mois sur un compte dédié pour lisser les pics de dépenses.\n2. **Cadrer les Télécommunications** : Identifiez la cause de la hausse de 73% et revenez au palier des 14 € pour stabiliser vos coûts fixes.\n3. **Créer un fonds de roulement** : Allouez tout surplus de revenus (lorsque vos revenus dépassent 1000 €) à une réserve de sécurité pour couvrir les mois où le solde est négatif.\n\n### Tendance\nLes postes "Famille" (-27%) et "Divertissement" (-56%) sont en baisse significative par rapport aux mois précédents, mais cela ne suffit pas à compenser le déficit structurel lié au loyer et à la volatilité de vos revenus.	2026-08-04 18:27:35.347688+00	1
\.


--
-- Data for Name: budgets; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.budgets (id, monthly_limit, created_at, domain_id, owner_id) FROM stdin;
3	150.00	2026-07-18 15:59:04.66369+00	24	1
5	200.00	2026-07-18 16:00:29.534534+00	26	1
6	50.00	2026-07-18 17:34:06.66885+00	30	1
7	200.00	2026-07-18 18:57:22.219284+00	27	1
8	150.00	2026-09-06 10:18:46.078026+00	23	1
\.


--
-- Data for Name: credits; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.credits (id, name, monthly_amount, months_total, start_month, note, owner_id) FROM stdin;
4	Impôts 2026-2025	1040.00	4	2026-09		1
\.


--
-- Data for Name: domains; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.domains (id, name, parent_id, color, owner_id, is_transfer) FROM stdin;
14	Transport	\N	#3F8F7B	\N	f
15	Alimentation	\N	#C77D2E	\N	f
16	Santé et Bien-être	\N	#D6455D	\N	f
17	Divertissement	\N	#8A5CD1	\N	f
18	Épargne et Investissements	\N	#2E8B57	\N	f
19	Vie Professionnelle	\N	#3D6B99	\N	f
21	Autres	\N	#8A8A99	\N	f
1	Logement	\N	#4C6FBF	\N	f
5	Famille	\N	#B5548A	\N	f
22	Logement	\N	#4C6FBF	1	f
23	Transport	\N	#3F8F7B	1	f
24	Alimentation	\N	#C77D2E	1	f
25	Santé et Bien-être	\N	#D6455D	1	f
26	Divertissement	\N	#8A5CD1	1	f
27	Épargne et Investissements	\N	#2E8B57	1	f
28	Vie Professionnelle	\N	#3D6B99	1	f
29	Famille	\N	#B5548A	1	f
30	Autres	\N	#8A8A99	1	f
32	Télécommunications	\N	#8C6D2F	1	f
33	Amis	\N	#A63A28	1	f
34	Logement	\N	#4C6FBF	2	f
35	Transport	\N	#3F8F7B	2	f
36	Alimentation	\N	#C77D2E	2	f
37	Santé et Bien-être	\N	#D6455D	2	f
38	Divertissement	\N	#8A5CD1	2	f
39	Épargne et Investissements	\N	#2E8B57	2	f
40	Vie Professionnelle	\N	#3D6B99	2	f
41	Famille	\N	#B5548A	2	f
42	Autres	\N	#8A8A99	2	f
43	Transfert interne	\N	#7A8290	2	t
31	Transfert interne	\N	#c51135	1	t
44	Investissements	\N	#d4ce11	1	f
\.


--
-- Data for Name: merchant_aliases; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.merchant_aliases (id, owner_id, match_key, display, created_at, is_pattern) FROM stdin;
1	1	Vers Ben-mohamed Madi T	Vers Trade Republic	2026-07-20 10:48:43.796667+00	f
2	1	Les Uli	Carrefour les Ulis	2026-07-20 10:49:00.07437+00	f
3	1	Lessonsharing Mtb- - -30f:1- - -1	Lessonsharing	2026-07-20 10:49:08.629226+00	f
4	1	Offre Essentiel	Frais Bancaire	2026-07-20 10:49:19.689856+00	f
5	1	Africa Exotique Les	Africa Exotique Les Ulis	2026-07-20 10:49:32.621604+00	f
6	1	Nyx Justeatemps Rois	Machine à café	2026-07-20 10:49:45.578101+00	f
7	1	Lessonsharing Mtb- - -22f:1- - -1	Lessonsharing	2026-07-20 10:49:51.168336+00	f
8	1	Vers Madi Toumani Sakin	Vers Sakina	2026-07-20 10:49:57.514441+00	f
9	1	Ulis Viandes Les Uli	Boucherie Les Ulis	2026-07-20 10:50:20.333257+00	f
10	1	Nyx Stcommerce Paris	Machine à café Paris	2026-07-20 10:50:34.102104+00	f
11	1	Vers Mme Jing An	Loyer	2026-07-20 10:51:04.173608+00	f
12	1	Vers M.madi Tarik	Vers Tarik	2026-07-20 10:51:11.468523+00	f
13	1	De Madi Toumani Sakina	De Sakina	2026-07-20 10:51:21.222186+00	f
14	1	De Mile Aya N’goran	De Sonia	2026-07-20 10:51:29.645563+00	f
15	1	De M. Madi Tarik	De Tarik	2026-07-20 10:51:39.749016+00	f
16	1	Snbru Air08223037157	Frais Brussels Airline	2026-07-20 10:52:02.071354+00	f
17	1	Snbru Air Roissy	Frais Brussels Airline	2026-07-20 10:52:12.882846+00	f
18	1	Vers Aya N’goran	Vers Sonia	2026-07-20 10:52:25.965093+00	f
19	1	Shein Dublin	Shein	2026-07-20 10:52:36.111427+00	f
20	1	De M. Madi Toumani Ben-moh	Vers Compte courant	2026-07-20 10:53:16.423305+00	f
21	1	Web M. Madi Toumani Ben-moha	Vers Livret A	2026-07-20 10:53:27.385787+00	f
22	1	Idfm Idfm - - Mand Mandate-reference00000000000055687 7fr42277457385 Core	IDF Mobilité	2026-07-20 11:02:29.894474+00	f
23	1	Acheel France Ref-927b2080281923c168gfgwbh4 Core	Assurance habitation Acheel France	2026-07-20 11:02:54.917849+00	f
24	1	Acheel France Ref-1cbbb9880770f24695gfgwbh4 Fr457z787f911 Core	Assurance habitation Acheel France	2026-07-20 11:03:09.649848+00	f
25	1	Acheel France Ref-2f3e4b269ac60316d9gfgwbh4 Core	Assurance habitation Acheel France	2026-07-20 11:03:12.489135+00	f
26	1	Acheel France Ref-996928ef59fa47a80bgfgwbh4 Fr4527787f911 Core	Assurance habitation Acheel France	2026-07-20 11:03:15.293355+00	f
27	1	Acheel France Ref-81821f2daf9edes013gfgwbh4 Core	Assurance habitation Acheel France	2026-07-20 11:03:18.190951+00	f
28	1	Tiktok Shop Seller D	Tiktok Shop	2026-07-20 11:03:38.946919+00	f
29	1	Acheel France Core	Assurance habitation Acheel France	2026-07-20 11:03:52.09711+00	f
30	1	Idfm Idfm - - Mand Ate-reference000000000000556877 Core	IDF Mobilité	2026-07-20 11:04:23.506352+00	f
31	1	Lessonsharing - - -mtb 00005f:1- - -1	Lessonsharing	2026-07-20 11:04:34.429489+00	f
32	1	Lessonsharing Mtb- - -30f:1- - -2	Lessonsharing	2026-07-20 11:04:35.225929+00	f
33	1	Lessonsharing - - -mtb 00002f:1- - -1	Lessonsharing	2026-07-20 11:04:35.985027+00	f
34	1	Lessonsharing - - -mtbf:1- - -1	Lessonsharing	2026-07-20 11:04:37.060676+00	f
35	1	Viandes Du Argent	Boucherie Argenteuil	2026-07-20 11:04:49.729333+00	f
36	1	Ornikar Paris	Ornikar Paris	2026-07-20 11:04:56.515321+00	f
37	1	Bouygues Telecom	Bouygues Telecom Mobile	2026-07-20 11:05:31.839914+00	f
38	1	Free	Free	2026-07-20 11:05:32.456856+00	f
39	1	Deliveroo	Deliveroo	2026-07-20 11:05:50.325233+00	f
40	1	Engie	Engie	2026-07-20 11:05:50.776179+00	f
41	1	Uber	Uber	2026-07-20 11:05:51.064414+00	f
42	1	Carrefour	Carrefour	2026-07-20 11:05:51.368797+00	f
43	1	Burger King	Burger King	2026-07-20 11:05:51.732512+00	f
44	1	Service Navigo Paris	Service Navigo Paris	2026-07-20 11:05:52.212893+00	f
45	1	PayPal	Paypal	2026-07-20 11:05:53.622144+00	f
46	1	Amazon	Amazon	2026-07-20 11:05:54.171227+00	f
47	1	SNCF	SNCF	2026-07-20 11:05:58.785647+00	f
48	1	Vers Assjad Saida	Vers Assjad	2026-07-20 11:06:06.794725+00	f
49	1	Royal Hotel Monrovia	Royal Hotel Monrovia	2026-07-20 11:06:19.465662+00	f
50	1	Vers Ambassade De Liber	Ambassade Du Liberia	2026-07-20 11:06:37.646946+00	f
51	1	Action	Action	2026-07-20 11:06:38.230361+00	f
52	1	Sc.pedra Alta Paris	Pedra Alta Paris	2026-07-20 11:06:51.027698+00	f
53	1	Ret Les Ulis 14h30	Retrait Distributeur Les Ulis	2026-07-20 11:07:07.38877+00	f
54	1	Ret Les Ulis 09h28	Retrait Distributeur Les Ulis	2026-07-20 11:07:10.751744+00	f
55	1	Heetch Transport Ba	Heetch	2026-07-20 11:07:55.70874+00	f
56	1	Five Pizza Les Ui	Five Pizza Les Uis	2026-07-20 11:07:56.203234+00	f
57	1	Heetch” Transport Ba	Heetch	2026-07-20 11:07:56.382661+00	f
58	1	Caresquick Topaze	Caresquick Topaze	2026-07-20 11:07:56.600568+00	f
59	1	La Toscana Paris	La Toscana Paris	2026-07-20 11:07:57.04389+00	f
60	1	Mcdo Lohearches Vinc	Mcdo	2026-07-20 11:07:57.807822+00	f
61	1	Yamanashi Paris	Yamanashi Paris	2026-07-20 11:07:58.882516+00	f
62	1	Sloann Paris	Sloann Paris	2026-07-20 11:08:00.382106+00	f
63	1	Lessonsharing Mtb- - -31f: - -1	Lessonsharing	2026-07-20 11:08:26.175813+00	f
64	1	Lessonsharing - - -mtb 00004f:1- - -1	Lessonsharing	2026-07-20 11:08:26.418386+00	f
65	1	Lessonsharing Mtb- - -31f:1- - -1	Lessonsharing	2026-07-20 11:08:26.610581+00	f
66	1	Lessonsharing Mtb- - -27f:1- - -1	Lessonsharing	2026-07-20 11:08:26.775853+00	f
67	1	Lessonsharing - - -mtb 00006f:1- - -1	Lessonsharing	2026-07-20 11:08:26.942678+00	f
68	1	Lessonsharing Mtb- - - F:1- - -1	Lessonsharing	2026-07-20 11:08:27.1177+00	f
69	1	Lessonsharing - - -mtb 00003f:1- - -1	Lessonsharing	2026-07-20 11:08:27.630238+00	f
70	1	Lessonsharing - - -1	LessonSharing	2026-07-21 07:24:52.34603+00	f
71	1	Lessonsharing - - - -1	LessonSharing	2026-07-21 07:24:54.075523+00	f
72	1	Snbru	SNBRU	2026-07-21 07:24:58.781582+00	f
73	1	Hippo Bercy Paris	Hippo Bercy	2026-07-21 07:25:01.665384+00	f
74	1	Wero Vers Aleksandar S	Wero Aleksandar	2026-07-21 07:25:03.718376+00	f
75	1	African Fashion F Pa	African Fashion	2026-07-21 07:25:04.58617+00	f
76	1	Idfm Idfm - - Mand Mandate- Core	IDFM Core	2026-07-21 07:25:06.704141+00	f
77	1	Wero Vers Vadijiguiba Di	Wero Vadijiguiba	2026-07-21 07:25:10.000432+00	f
78	1	Pizza Andiamo Les Ui	Andiamo Pizza	2026-07-21 07:25:12.480504+00	f
79	1	Food Les Ulis	Les Ulis Food	2026-07-21 07:25:14.060848+00	f
80	1	Idfm Idfm - - Mand Ate- Core	IDFM Core	2026-07-21 07:25:15.322665+00	f
81	1	Ret Les Ulis	Retrait Distributeur Les Ulis	2026-07-21 07:25:36.11573+00	f
82	1	Lessonsharing - - -2	Lessonsharing	2026-07-21 07:26:06.652186+00	f
83	1	Phcie Courdimanc Les	Phcie Courdimanc	2026-07-21 07:26:07.419984+00	f
84	1	X5 Coinbase Ireland Dub	Coinbase	2026-07-30 20:25:31.60086+00	f
85	1	X5 Coinbase Dub	Coinbase	2026-07-30 20:25:34.856699+00	f
86	1	Vers Trade Republic	Vers Trade Republic	2026-07-30 20:25:45.389423+00	f
87	1	Vers Sakina Madi Touman	Vers Sakina	2026-07-30 20:25:58.907351+00	f
88	1	1 . Vers Abdou Madi Toumani	Vers Abdou	2026-07-30 20:26:04.586367+00	f
89	1	. Vers Sakina Madi Touman	Vers Sakina	2026-07-30 20:26:10.818328+00	f
90	1	Airbnb	Airbnb	2026-07-30 20:26:15.588784+00	f
91	1	Vers Lessonsharing	Vers Lessonsharing	2026-07-30 20:26:23.458476+00	f
92	1	Instant Gaming Dubli	Instant Gaming	2026-07-30 20:26:28.636323+00	f
93	1	RATP	RATP	2026-07-30 20:26:32.056568+00	f
94	1	Pharmacie Ulis 2 Les	Pharmacie Les Ulis	2026-07-30 20:26:48.323457+00	f
95	1	La Poste	La Poste	2026-07-30 20:26:52.132434+00	f
96	1	Coinbase Dublin	Coinbase	2026-07-30 20:26:55.652324+00	f
97	1	Coinbase Ireland Dub	Coinbase	2026-07-30 20:26:57.955315+00	f
98	1	Vers Abdou	Vers Abdou	2026-07-30 20:27:02.720008+00	f
99	1	Binance.com Vilnius	Binance	2026-07-30 20:27:07.737117+00	f
100	1	Vers Arielle Rev	Vers Arielle	2026-07-30 20:27:14.398498+00	f
101	1	Lessonsharing - -	Lessonsharing	2026-07-30 20:27:18.316158+00	f
102	1	Temu.com Dublin 2	Temu	2026-07-30 20:27:23.43187+00	f
103	1	Vers Assjad	Vers Assjad	2026-07-30 20:27:27.538802+00	f
104	1	Vers Ericia Arielle Rev	Vers Arielle	2026-07-30 20:28:07.646487+00	f
105	1	L Empire Du Costume	L'Empire Du Costume	2026-07-30 20:29:10.111676+00	f
106	1	Vers Trade Republic Pea	Vers Trade Republic	2026-07-30 20:30:18.366539+00	f
107	1	Trade Republic Berli	Vers Trade Republic	2026-07-30 20:30:32.494088+00	f
108	1	. Vers Ben-mohamed Madi T	Vers Trade Republic	2026-07-30 20:30:43.381308+00	f
109	1	X Temu.com Dublin 2	Temu	2026-07-30 20:30:49.753126+00	f
110	1	Prestige Kebab	Prestige Kebab	2026-07-30 20:31:10.564325+00	f
111	1	Sushi One Les Ulis	Sushi One Les Ulis	2026-07-30 20:31:16.897966+00	f
112	1	Lh Boucherie Les Uli	Boucherie Les Ulis	2026-07-30 20:31:23.159148+00	f
113	1	X5 Photomaton Echirolle	Photomaton Echirolle	2026-07-30 20:31:30.359255+00	f
114	1	Ulis Kitchen Les Uli	Kitchen Les Ulis	2026-07-30 20:32:50.367473+00	f
115	1	Chamas Tacos Marseil	Chamas Tacos	2026-07-30 20:32:57.595507+00	f
116	1	X5 Pizza Andiamo Les Ul	Pizza Andiamo Les Ulis	2026-07-30 20:33:06.450396+00	f
117	1	X Phcie Courdimanc Les	Pharmacie Courdimanc	2026-07-30 20:33:16.946622+00	f
118	1	O’delices Malakof	O’delices Malakof	2026-07-30 20:33:21.455097+00	f
119	1	Gc Re Keepcool Keep Core	Keepcool	2026-07-30 20:33:29.81923+00	f
120	1	Timbre Fiscal Ren	Timbre Fiscal	2026-07-30 20:33:38.627004+00	f
121	1	Cleor Les Ulis	Cleor	2026-07-30 20:33:43.890852+00	f
122	1	Lp Photocopieurs Lyo	Photocopieurs	2026-07-30 20:33:48.846427+00	f
123	1	Nya Juste Temps Pa	Machine à café	2026-07-30 21:57:20.457647+00	f
124	1	Nya Juste A Temps Pa	Machine à café	2026-07-30 21:57:25.092023+00	f
125	1	U Express Lyon	U Express	2026-07-30 21:57:39.702714+00	f
126	1	Wakanda Factory Pari	Wakanda Factory Paris	2026-07-30 21:57:44.444271+00	f
127	1	Titanic Massy	Titanic Massy	2026-07-30 21:57:48.953667+00	f
128	1	X5 Chatelet Resto Paris	Chatelet Resto Paris	2026-07-30 21:57:54.556152+00	f
129	1	Intérets Débiteurs Période 2 Trim Taeg= , %	Intérets Débiteurs	2026-07-30 21:58:04.016405+00	f
130	1	Intérets Débiteurs Période 1 Trim Taeg= , %	Intérets Débiteurs	2026-07-30 21:58:06.056112+00	f
131	1	Lessonsharing Salaire Février	Lessonsharing Salaire Février	2026-07-30 21:58:14.527121+00	f
132	1	Lessonsharing Facture 1- - -2	Lessonsharing	2026-07-30 21:58:18.777734+00	f
133	1	De Acheel France	Acheel France	2026-07-30 21:58:24.524562+00	f
134	1	Megdiche Marseille	Megdiche Marseille	2026-07-30 21:58:32.036192+00	f
135	1	Web Clement Depernet	Clement Depernet	2026-07-30 21:58:36.980017+00	f
136	1	Sodexo Epicerie Pari	Sodexo Epicerie Paris	2026-07-30 21:58:54.778556+00	f
137	1	Sodexo Coeur De Pari	Sodexo Coeur De Paris	2026-07-30 21:59:05.350784+00	f
138	1	Buy Paris Duty Roiss	Buy Paris Duty Roiss	2026-07-30 21:59:14.212926+00	f
139	1	Ret Paris Chateau	Retrait Distributeur	2026-07-30 21:59:27.930876+00	f
140	1	X Maison Nouilles Pari	Nouilles Paris	2026-07-31 11:27:53.176861+00	f
141	1	Smile World Malakoff	Smile World	2026-07-31 11:27:55.94465+00	f
142	1	Germanho Multi- Les	Germanho Multi	2026-07-31 11:27:57.900718+00	f
143	1	Caresquick Zaventem	Caresquick	2026-07-31 11:28:16.843649+00	f
144	1	X Boucherie Marche Les	Boucherie Marche	2026-07-31 11:28:21.26113+00	f
145	1	Sc-elyians Les Ulis	Les Ulis Sc-Elyans	2026-07-31 11:28:23.699319+00	f
146	1	T	T	2026-07-31 11:28:26.203758+00	f
147	1	Mihir Et Associ Les	Mihir Et Assoc	2026-07-31 11:28:29.759619+00	f
148	1	Web Madi Toumani	Vers Livret A	2026-07-31 11:29:29.885224+00	f
149	1	De Madi Toumani	Vers Compte-courant	2026-07-31 11:30:42.405516+00	f
150	1	Ret Cesson Sevigne	Retrait Distributeur Cesson	2026-07-31 11:34:41.639704+00	f
151	1	De M. Madi Toumani	Vers Compte courant	2026-07-31 11:35:44.742934+00	f
152	1	1 . Web M. Madi Toumani Ben-moha	Vers Livret A	2026-07-31 11:36:21.906522+00	f
153	1	1 . De M. Madi Toumani Ben-moh	Vers Compte courant	2026-07-31 11:36:44.233362+00	f
154	1	Ret Les Ulis 2 Ga	Retrait Distributeur  Les Ulis	2026-07-31 11:37:04.95706+00	f
155	1	Paris	Machine à café	2026-07-31 11:37:31.927042+00	f
156	1	Les Ulis	Carrefour Les Ulis	2026-07-31 11:38:10.272027+00	f
157	1	Arcueil	Gare Arcueil	2026-07-31 11:38:23.512844+00	f
\.


--
-- Data for Name: rules; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.rules (id, pattern, domain_id, owner_id, tags) FROM stdin;
57	Amazon Payments	26	1	amazon
7	ENGIE	1	\N	
8	EDF	1	\N	
9	LOYER	1	\N	
59	M. Madi Tarik	29	1	frere
23	SALAIRE	19	\N	
14	UBER EATS	17	\N	
15	DELIVEROO	17	\N	
16	MCDONALD	17	\N	
17	ORANGE	17	\N	
18	FREE MOBILE	17	\N	
19	AMAZON PRIME	17	\N	
20	SPOTIFY	17	\N	
21	NETFLIX	17	\N	
1	MONOPRIX	21	\N	
2	INTERMARCHE	21	\N	
3	LIDL	21	\N	
4	AUCHAN	21	\N	
5	LECLERC	21	\N	
6	CARREFOUR	21	\N	
10	re:\\yESSO\\y	21	\N	
11	TOTALENERGIES	21	\N	
12	UBER	21	\N	
13	SNCF	21	\N	
22	PHARMACIE	21	\N	
24	VIREMENT	21	\N	
25	FRAIS	21	\N	
26	COTISATION	21	\N	
27	CARREFOUR	24	1	
30	LIDL	24	1	
32	MONOPRIX	24	1	
34	CASINO	24	1	
35	LOYER	22	1	
38	VEOLIA	22	1	
48	DOCTEUR	25	1	
49	FRAIS BANCAIRES	30	1	
50	COTISATION	30	1	
53	Vir Inst vers Madi Toumani Sakin	29	1	soeur
54	Prlv Engie Mandat Core	22	1	facture,abonnement,energie,engie
65	Mile Aya N’goran	33	1	
70	LESSONSHARING	28	1	salaire,lessonsharing
72	Hotel Monrovia	29	1	voyage,mariage,jessica
73	Vir Inst de Madi Toumani Sakina	29	1	soeur
71	TARIK	29	1	frere
61	Nyx*stcommerce Paris	28	1	cafe,ipssi
74	PAYPAL	26	1	paypal
76	COURDIMANC	25	1	
77	ASSJAD	29	1	niece
78	GORAN	33	1	
31	Carte X4519 Action Les Ulis	24	1	courses,action
42	RATP	23	1	ratp
55	Nyx*justeatemps Rois	28	1	cafe,ipssi
64	X4519 Les Uli	24	1	courses,carrefour
58	URSSAF	28	1	urssaf,taxes,cotisations
43	Web M. Madi Toumani Ben-moha	31	1	deplacement,livret-a
28	Carte X4519 Ulis Viandes Les Uli	24	1	courses,boucherie
80	Prlv Idfm IDFM	23	1	navigo
52	Vir Inst vers Ben-mohamed Madi T	29	1	trade-republic
62	Prlv Offre Essentiel	30	1	abonnement,frais-bancaires,credit agricole
63	Prlv Free Telecom Free Hautdebit	22	1	facture,abonnement,internet,free
51	De M. Madi Toumani Ben-moh	31	1	deplacement,compte-courant
45	Prlv Paypal (europe) $.a r.l. et Cie	26	1	paypal,credit
46	Carte X4519 Amazon	26	1	amazon
66	Mme Jing An	22	1	loyer
68	DELIVEROO	24	1	fast-food,deliveroo
41	Virement De M. Madi Toumani Ben-moh	31	1	deplacement,compte-courant
60	Acheel France	22	1	facture,abonnement,assurance,acheel
56	Prlv Bouygues Telecom	32	1	facture,abonnement,mobile,bouygues
69	Mens.pel	27	1	pel
79	Burger King	24	1	fast-food,burger-king
67	Service Navigo	23	1	navigo
40	UBER	23	1	uber
75	Africa Exotique	24	1	courses,africain
82	CARREFOUR	36	2	
83	LECLERC	36	2	
84	AUCHAN	36	2	
85	LIDL	36	2	
86	INTERMARCHE	36	2	
87	MONOPRIX	36	2	
88	FRANPRIX	36	2	
89	CASINO	36	2	
90	LOYER	34	2	
91	EDF	34	2	
92	ENGIE	34	2	
93	VEOLIA	34	2	
94	SNCF	35	2	
95	UBER	35	2	
96	TOTALENERGIES	35	2	
97	RATP	35	2	
98	ESSENCE	35	2	
99	NETFLIX	38	2	
100	SPOTIFY	38	2	
101	DISNEY	38	2	
102	PHARMACIE	37	2	
103	DOCTEUR	37	2	
104	FRAIS BANCAIRES	42	2	
105	COTISATION	42	2	
107	SEVIGNE	30	1	
47	PHARMACIE	25	1	pharmacie
110	PARIS	28	1	
111	VIANDES	24	1	
112	Carrefour les Ulis	24	1	courses,carrefour
33	De Madi Toumani	31	1	deplacement,compte-courant
109	Nya*juste	28	1	cafe,ipssi
114	TITANIC	24	1	
36	EDF	22	1	edf
44	NETFLIX	26	1	netflix,abonnement
117	DUBLIN	26	1	temu
120	ESSONNE	30	1	
116	ARIELLE	30	1	logement
113	ABDOU	29	1	frere
108	SAKINA	29	1	soeur
119	AIRBNB	22	1	airbnb
29	De M. Madi Toumani	31	1	deplacement,compte-courant
122	Web Madi Toumani	31	1	deplacement,livret-a
123	EXPRESS	24	1	
124	DELICES	24	1	
126	WAKANDA	24	1	
128	MALAKOFF	33	1	
115	REPUBLIC	29	1	trade-republic
106	MONROVIA	29	1	jessica,voyage,voyage-liberia
121	COINBASE	44	1	crypto,coinbase
130	BINANCE	44	1	crypto,binance
132	LES Uli	24	1	courses,carrefour
135	Avoir Carte X5781 Amazon Payment	30	1	amazon,remboursement
39	SNCF	23	1	sncf
125	KEEPCOOL	25	1	keepcool,abonnement,salle-de-sport
37	ENGIE	22	1	engie
127	PHOTOCOPIEURS	30	1	administratif
118	PHOTOMATON	30	1	administratif
\.


--
-- Data for Name: schema_migrations; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.schema_migrations (name, applied_at) FROM stdin;
001_init.sql	2026-07-18 13:33:35.35836+00
002_budgets.sql	2026-07-18 13:33:35.376151+00
003_settings.sql	2026-07-18 13:33:35.379833+00
004_features.sql	2026-07-18 13:33:35.382182+00
005_domains_seed.sql	2026-07-18 13:33:35.404818+00
006_domain_taxonomy.sql	2026-07-18 13:33:35.405641+00
007_domains_dedup.sql	2026-07-18 13:33:35.410821+00
008_categories.sql	2026-07-18 13:33:35.413077+00
009_rules_unique.sql	2026-07-18 13:33:35.414855+00
010_data_isolation.sql	2026-07-18 13:33:35.416429+00
011_budget_owner_unique.sql	2026-07-18 13:33:35.432647+00
012_rule_tags.sql	2026-07-18 13:33:35.434214+00
013_transfer_category.sql	2026-07-18 13:33:35.434893+00
014_display_labels.sql	2026-07-20 09:55:19.603055+00
015_label_kind.sql	2026-07-20 09:55:19.606142+00
016_merchant_aliases.sql	2026-07-20 09:55:19.607042+00
017_tag_pin.sql	2026-07-20 09:55:19.610917+00
018_alias_pattern.sql	2026-07-21 07:21:03.753969+00
\.


--
-- Data for Name: sessions; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.sessions (token, user_id, expires_at) FROM stdin;
ssyhBCGHNrAqp9_8nRZ_6cIx81gkYX7P9As8Xbh2lSw	1	2026-10-06 09:43:03.782663+00
\.


--
-- Data for Name: settings; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.settings (key, value, updated_at, owner_id) FROM stdin;
dashboard	[{"id": "w1", "size": "full", "type": "kpi", "config": {}}, {"id": "w2", "size": "full", "type": "flux", "config": {"months": 12}}, {"id": "wa", "size": "half", "type": "attention", "config": {}}, {"id": "w4", "size": "half", "type": "solde", "config": {"months": 12}}, {"id": "w100_2989", "size": "half", "type": "trend", "config": {"months": 12, "category": "Investissements"}}, {"id": "w3", "size": "half", "type": "categories", "config": {}}, {"id": "w5", "size": "half", "type": "top", "config": {"count": 5}}, {"id": "w6", "size": "half", "type": "subs", "config": {}}]	2026-07-31 10:45:39.303131+00	1
ai_config	{"context": "Je paie mon loyer de manière trimestriel.\\nJe suis indépendant mes revenus varient énormément.", "provider": "ollama", "ollama_url": "http://192.168.1.180:11434", "ollama_model": "gemma4:12b", "anthropic_model": "claude-sonnet-4-6", "auto_categorize": true}	2026-08-04 18:26:27.371063+00	\N
\.


--
-- Data for Name: statements; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.statements (id, filename, period_start, period_end, opening_balance, closing_balance, parser, validated, uploaded_at, account_id, owner_id) FROM stdin;
1	Releve n001 du 21-01-2026-CCHQ 43609805111 M MADI TOUMANI.pdf	2025-12-22	2026-01-21	68.18	611.81	local	t	2026-07-18 13:34:23.07909+00	1	1
2	Releve n002 du 20-02-2026-CCHQ 43609805111 M MADI TOUMANI.pdf	2026-01-21	2026-02-20	611.81	112.46	local	f	2026-07-18 14:42:58.237521+00	1	1
3	Releve n003 du 20-03-2026-CCHQ 43609805111 M MADI TOUMANI.pdf	2026-02-20	2026-03-20	112.46	208.71	local	t	2026-07-18 15:41:31.90789+00	1	1
4	Releve n004 du 21-04-2026-CCHQ 43609805111 M MADI TOUMANI.pdf	2026-03-20	2026-04-21	208.71	53.02	local	t	2026-07-18 15:44:49.806446+00	1	1
5	Releve n005 du 21-05-2026-CCHQ 43609805111 M MADI TOUMANI.pdf	2026-04-21	2026-05-21	53.02	306.46	local	t	2026-07-18 15:45:25.873851+00	1	1
7	Releve n006 du 22-06-2026-CCHQ 43609805111 M MADI TOUMANI.pdf	2026-05-21	2026-06-22	306.46	38.47	local	t	2026-07-18 18:17:23.945522+00	1	1
8	Relevé n°007 du 21-07-2026_CCHQ 43609805111 M MADI TOUMANI.pdf	2026-06-22	2026-07-21	38.47	121.65	local	t	2026-07-28 22:50:55.361914+00	1	1
10	Relevé n°012 du 22-12-2025_CCHQ 43609805111 M MADI TOUMANI.pdf	2025-11-21	2025-12-22	58.09	68.18	local	t	2026-07-30 19:38:32.975416+00	1	1
11	Relevé n°011 du 21-11-2025_CCHQ 43609805111 M MADI TOUMANI.pdf	2025-10-21	2025-11-21	148.91	58.09	local	f	2026-07-30 19:40:28.935212+00	1	1
12	Relevé n°010 du 21-10-2025_CCHQ 43609805111 M MADI TOUMANI.pdf	2025-09-22	2025-10-21	95.10	148.91	local	t	2026-07-30 19:41:56.491763+00	1	1
13	Relevé n°009 du 22-09-2025_CCHQ 43609805111 M MADI TOUMANI.pdf	2025-08-21	2025-09-22	99.26	95.10	local	t	2026-07-30 19:50:21.824998+00	1	1
14	Relevé n°008 du 21-08-2025_CCHQ 43609805111 M MADI TOUMANI.pdf	2025-07-21	2025-08-21	113.86	99.26	local	f	2026-07-30 19:52:11.555021+00	1	1
15	Relevé n°007 du 21-07-2025_CCHQ 43609805111 M MADI TOUMANI.pdf	2025-06-20	2025-07-21	594.97	113.86	local	t	2026-07-30 19:53:35.561533+00	1	1
16	Relevé n°006 du 20-06-2025_CCHQ 43609805111 M MADI TOUMANI.pdf	2025-05-21	2025-06-20	128.12	594.97	local	t	2026-07-30 19:57:17.4125+00	1	1
17	Relevé n°004 du 22-04-2025_CCHQ 43609805111 M MADI TOUMANI.pdf	2025-03-21	2025-04-22	101.19	61.78	local	f	2026-07-30 19:59:09.120187+00	1	1
18	Relevé n°003 du 21-03-2025_CCHQ 43609805111 M MADI TOUMANI.pdf	2025-02-21	2025-03-21	25.37	101.19	local	f	2026-07-30 20:01:29.345397+00	1	1
20	Relevé n°002 du 21-02-2025_CCHQ 43609805111 M MADI TOUMANI.pdf	2025-01-21	2025-02-21	102.97	25.37	local	f	2026-07-30 20:05:19.623843+00	1	1
21	Releve n008 du 21-08-2026-CCHQ 43609805111 M MADI TOUMANI.pdf	2026-07-21	2026-08-21	121.65	2492.88	local	t	2026-09-06 09:49:30.32975+00	1	1
\.


--
-- Data for Name: tag_folder_prefs; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.tag_folder_prefs (owner_id, tag_name, state) FROM stdin;
1	taxes	hidden
1	deliveroo	hidden
1	courses	hidden
1	erreur	hidden
\.


--
-- Data for Name: tags; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.tags (id, name, owner_id) FROM stdin;
119	move	1
82	hotel	1
81	transfert	1
86	pizza	1
199	frigo	1
373	alimentation	1
67	taxi	1
1846	pere	1
78	soins	1
1796	cofidis	1
1860	medicaments	1
234	tiktok	1
638	starlink	1
237	visa	1
1835	erreur	1
1899	demenagement	1
1966	horace	1
2518	repas	1
15532	materiels	1
1794	viandes	1
13014	cloud	1
184	permis	1
15458	courriel	1
637	projet	1
1964	pea	1
15472	bourse	1
240	shein	1
15845	vetements	1
16289	paris	1
227	restaurant	1
329	coiffeur	1
16405	instant-gaming	1
16406	jeux	1
65	boucherie	1
13006	sncf	1
18362	demenagement-toulon	1
64	courses	1
70	navigo	1
262	carrefour	1
18373	laboratoire	1
331	burger-king	1
30	mobile	1
24	cafe	1
18375	ip	1
31	bouygues	1
16	energie	1
245	uber	1
1969	caf	1
18369	amis	1
62	internet	1
63	free	1
16372	keepcool	1
15	abonnement	1
355	action	1
3827	ratp	1
71	fast-food	1
1283	africain	1
68	loyer	1
14	facture	1
1	deplacement	1
19	compte-courant	1
87	salaire	1
88	lessonsharing	1
66	remboursement	1
235	cadeau	1
72	deliveroo	1
16374	salle-de-sport	1
122	livret-a	1
13	soeur	1
15558	airbnb	1
172	mariage	1
369	pret	1
75	pel	1
39	assurance	1
40	acheel	1
59	frais-bancaires	1
179	credit agricole	1
1863	administratif	1
13021	pharmacie	1
32	amazon	1
187	paypal	1
45	ipssi	1
204	credit	1
171	voyage	1
1948	voyage-liberia	1
173	jessica	1
183	niece	1
17	engie	1
15540	logement	1
16046	coinbase	1
13018	crypto	1
13017	binance	1
16393	edf	1
33	urssaf	1
34	taxes	1
35	cotisations	1
36	frere	1
6	trade-republic	1
16397	temu	1
\.


--
-- Data for Name: taxes; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.taxes (id, name, amount, due_date, paid, owner_id) FROM stdin;
4	cotisation urssaf aout	775.00	2026-09-30	f	1
5	prêt Tarik	800.00	2026-10-10	f	1
6	prêt fazal	2300.00	2026-10-04	f	1
\.


--
-- Data for Name: transaction_tags; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.transaction_tags (transaction_id, tag_id) FROM stdin;
46	1
16	1
45	6
40	6
44	13
43	14
43	15
43	16
43	17
42	1
42	19
39	1
39	19
38	1
38	19
37	24
31	24
34	24
35	24
36	14
36	15
36	30
36	31
33	32
32	33
32	34
32	35
30	36
28	14
28	15
28	39
28	40
27	24
17	24
24	24
37	45
31	45
34	45
35	45
27	45
17	45
24	45
26	15
26	59
23	14
23	15
23	62
23	63
20	67
18	68
14	70
13	71
13	72
12	71
12	72
11	75
50	67
51	67
83	82
84	82
101	86
1	87
1	88
2	87
2	88
3	87
3	88
4	87
4	88
5	87
5	88
78	87
78	88
69	87
69	88
70	87
70	88
71	87
71	88
72	87
72	88
73	87
73	88
74	87
74	88
75	87
75	88
76	87
76	88
77	87
77	88
82	122
82	1
16	19
54	1
54	19
97	1
97	19
98	1
98	19
108	1
108	19
46	19
58	1
58	19
66	1
66	19
110	1
110	19
111	1
111	19
68	75
83	171
83	172
83	173
84	171
84	172
84	173
26	179
88	15
88	59
88	179
90	183
90	184
118	71
164	187
167	81
200	187
206	67
207	187
212	86
228	13
228	199
81	36
164	204
207	204
56	187
56	204
167	187
213	187
217	187
208	187
211	227
119	227
177	227
115	227
55	187
55	66
121	245
120	245
106	33
106	34
106	35
105	33
105	34
105	35
104	14
104	15
104	16
104	17
102	70
52	64
52	262
100	64
100	262
140	64
140	262
171	64
171	262
195	64
195	262
224	64
224	262
226	64
226	262
219	64
219	65
218	64
204	71
204	331
196	64
196	65
182	64
182	355
103	64
103	65
47	64
47	355
22	64
22	262
166	13
166	369
80	13
80	373
124	87
124	88
173	87
173	88
174	87
174	88
191	33
191	34
191	35
150	16
150	14
150	15
150	17
201	16
201	14
201	15
201	17
227	16
227	14
227	15
227	17
92	14
92	15
92	30
92	31
141	14
141	15
141	30
141	31
198	14
198	15
198	30
198	31
225	14
225	15
225	30
225	31
63	6
67	6
87	6
94	6
144	6
96	32
163	32
176	32
184	32
210	32
210	66
203	32
203	66
183	32
116	1
116	19
132	1
132	19
136	1
136	19
142	1
142	19
143	1
143	19
148	1
148	19
151	1
151	19
154	1
154	19
155	1
155	19
159	1
159	19
165	1
165	19
168	1
168	19
170	1
170	19
179	1
179	19
192	1
192	19
199	1
199	19
202	1
202	19
209	1
209	19
133	15
133	59
133	179
190	15
190	59
190	179
223	15
223	59
223	179
126	1
126	122
158	1
158	122
85	14
85	15
85	62
85	63
127	14
127	15
127	62
127	63
189	14
189	15
189	62
189	63
222	14
222	15
222	62
222	63
180	68
86	14
86	15
86	39
86	40
130	14
130	15
130	39
130	40
188	14
188	15
188	39
188	40
220	14
220	15
220	39
220	40
123	75
175	75
215	75
161	70
7	245
8	245
206	70
129	71
129	72
134	71
134	72
152	71
152	72
160	71
160	72
185	71
185	72
49	64
49	1283
139	64
139	1283
194	64
194	1283
99	64
99	1283
48	227
10	369
10	66
285	14
285	15
285	62
285	63
289	1
289	122
299	1
299	19
308	1
308	19
310	14
310	15
310	16
310	17
306	14
306	15
306	30
306	31
297	15
297	59
297	179
272	70
279	87
279	88
280	87
280	88
281	87
281	88
282	87
282	88
283	87
283	88
284	87
284	88
277	36
303	64
303	262
278	14
278	15
278	39
278	40
271	71
271	331
291	71
291	331
276	75
307	183
293	33
293	34
293	35
311	33
311	34
311	35
287	36
312	32
312	1796
312	13
305	234
305	235
305	173
304	187
304	66
301	187
301	204
301	171
301	173
300	32
300	173
197	204
197	187
197	171
197	173
205	329
300	637
300	638
298	329
296	237
296	171
296	173
292	240
292	235
292	173
290	235
205	262
205	64
214	204
214	187
214	1835
295	204
295	187
295	171
295	173
295	1835
62	245
62	1835
146	1846
147	1846
294	1794
294	65
275	64
275	262
274	64
274	262
128	64
128	262
59	78
59	173
59	235
53	171
53	1860
157	173
157	172
157	1863
21	66
21	369
60	1835
60	171
60	173
61	1835
61	171
61	173
9	6
9	173
29	6
29	173
41	6
41	173
64	6
64	173
57	6
57	173
79	6
79	173
89	6
89	173
91	6
91	173
95	6
95	173
109	6
109	173
109	637
109	1899
112	6
112	173
112	637
112	1899
117	6
117	173
122	6
122	173
125	6
125	173
131	6
131	173
135	6
135	173
137	6
137	173
138	6
138	173
149	6
149	173
162	6
162	173
169	6
169	173
172	6
172	173
178	6
178	173
186	6
186	173
193	6
193	173
216	6
216	173
286	6
286	173
302	6
302	173
309	6
309	173
309	1948
309	171
288	6
288	173
288	1948
288	171
15	32
15	6
19	6
19	32
93	6
93	1966
107	6
107	34
107	1969
337	187
337	204
313	1
313	19
341	1
341	19
334	13
338	13
331	14
331	15
331	62
331	63
320	64
320	355
319	1
319	19
325	1
325	19
328	1
328	19
329	1
329	19
343	14
343	15
343	16
343	17
339	14
339	15
339	30
339	31
336	15
336	59
336	179
344	70
298	64
298	262
317	64
317	262
322	64
322	262
324	64
324	262
333	14
333	15
333	39
333	40
330	68
327	75
323	245
323	1948
321	173
321	235
321	1948
318	235
316	262
316	2518
314	262
314	2518
378	1
378	19
380	1
380	19
403	1
403	19
410	1
410	19
416	1
416	19
424	1
424	19
387	13
407	13
419	13
409	64
409	65
386	6
398	6
423	6
426	6
393	14
393	15
393	62
393	63
377	1
377	19
384	1
384	19
390	1
390	19
394	1
394	19
399	1
399	19
421	1
421	19
420	14
420	15
420	16
420	17
415	14
415	15
415	30
415	31
400	15
400	59
400	179
408	64
408	1283
388	70
385	87
385	88
382	64
382	262
425	64
425	262
402	14
402	15
402	39
402	40
381	71
381	331
404	71
404	331
395	71
395	72
389	75
413	245
456	187
456	204
442	1
442	19
478	1
478	19
432	6
444	6
458	6
463	14
463	15
463	62
463	63
459	1
459	122
460	1
460	122
476	1
476	122
436	64
436	355
433	1
433	19
445	1
445	19
468	1
468	19
477	14
477	15
477	16
477	17
474	14
474	15
474	30
474	31
466	15
466	59
466	179
455	70
446	87
446	88
447	87
447	88
448	87
448	88
449	87
449	88
450	87
450	88
451	87
451	88
452	87
452	88
453	87
453	88
454	64
454	262
465	14
465	15
465	39
465	40
469	71
469	72
443	75
461	33
461	34
461	35
462	33
462	34
462	35
429	3827
430	3827
440	245
441	245
514	187
514	204
519	1
519	19
520	1
520	19
524	1
524	19
516	14
516	15
516	62
516	63
491	1
491	19
497	1
497	19
503	1
503	19
512	1
512	19
529	1
529	19
528	14
528	15
528	16
528	17
526	14
526	15
526	30
526	31
483	24
483	45
518	15
518	59
518	179
482	64
482	1283
499	70
504	87
504	88
505	87
505	88
506	87
506	88
507	87
507	88
508	87
508	88
521	64
521	262
517	14
517	15
517	39
517	40
510	68
525	71
525	72
509	75
500	33
500	34
500	35
561	187
561	204
536	13
559	13
562	14
562	15
562	62
562	63
558	1
558	122
533	1
533	19
535	1
535	19
537	1
537	19
544	1
544	19
545	1
545	19
548	1
548	19
560	1
560	19
578	14
578	15
578	16
578	17
567	14
567	15
567	30
567	31
564	15
564	59
564	179
551	64
551	1283
570	64
570	1283
554	70
549	87
549	88
550	87
550	88
566	14
566	15
566	39
566	40
538	71
538	331
565	71
565	331
556	75
608	187
608	204
631	187
631	204
580	1
580	19
617	1
617	19
619	1
619	19
625	1
625	19
610	14
610	15
610	62
610	63
591	1
591	122
592	1
592	122
596	1
596	19
597	1
597	19
612	1
612	19
614	1
614	19
627	1
627	19
635	1
635	19
636	1
636	19
630	14
630	15
630	16
630	17
623	14
623	15
623	30
623	31
615	15
615	59
615	179
601	64
601	1283
582	70
586	87
586	88
587	87
587	88
588	87
588	88
611	14
611	15
611	39
611	40
603	71
603	331
609	71
609	331
594	75
621	33
621	34
621	35
595	36
613	36
691	187
691	204
674	1
674	19
675	1
675	19
686	1
686	19
689	1
689	19
670	14
670	15
670	62
670	63
639	1
639	19
640	1
640	19
641	1
641	19
642	1
642	19
649	1
649	19
654	1
654	19
656	1
656	19
657	1
657	19
663	1
663	19
666	1
666	19
669	1
669	19
672	1
672	19
687	14
687	15
687	16
687	17
685	14
685	15
685	30
685	31
681	15
681	59
681	179
650	32
676	32
661	70
680	14
680	15
680	39
680	40
668	68
653	71
653	331
646	71
646	72
662	75
648	33
648	34
648	35
643	36
671	36
684	36
744	187
744	204
693	1
693	19
742	1
742	19
722	13
726	14
726	15
726	62
726	63
717	1
717	122
723	1
723	122
692	1
692	19
694	1
694	19
729	1
729	19
732	1
732	19
745	1
745	19
743	14
743	15
743	16
743	17
741	14
741	15
741	30
741	31
730	15
730	59
730	179
733	64
733	1283
720	70
708	87
708	88
709	87
709	88
710	87
710	88
711	87
711	88
712	87
712	88
719	87
719	88
727	14
727	15
727	39
727	40
704	71
704	72
736	71
736	72
718	75
707	183
728	183
713	36
805	1
805	19
777	14
777	15
777	62
777	63
752	1
752	19
757	1
757	19
759	1
759	19
775	1
775	19
780	1
780	19
787	1
787	19
789	1
789	19
794	1
794	19
796	1
796	19
797	1
797	19
803	1
803	19
808	1
808	19
812	1
812	19
807	14
807	15
807	16
807	17
802	14
802	15
802	30
802	31
784	15
784	59
784	179
760	64
760	1283
785	64
785	1283
762	70
764	87
764	88
783	14
783	15
783	39
783	40
778	68
771	71
771	331
767	75
748	187
749	187
766	187
758	183
773	33
773	34
773	35
793	36
765	245
849	187
849	204
848	14
848	15
848	62
848	63
875	14
875	15
875	16
875	17
869	14
869	15
869	30
869	31
852	15
852	59
852	179
851	32
855	32
863	32
865	32
866	32
818	64
818	1283
826	70
830	87
830	88
831	87
831	88
837	87
837	88
841	14
841	15
841	39
841	40
857	71
857	72
829	75
832	187
853	187
854	187
864	187
881	187
827	183
836	33
836	34
836	35
859	245
860	245
924	187
924	204
899	13
927	14
927	15
927	62
927	63
889	14
889	15
889	16
889	17
941	14
941	15
941	16
941	17
939	14
939	15
939	30
939	31
930	15
930	59
930	179
907	32
916	64
916	1283
902	70
903	70
914	87
914	88
942	14
942	15
942	39
942	40
928	71
928	331
912	75
906	187
934	187
897	33
897	34
897	35
922	33
922	34
922	35
6	227
6	235
427	6
427	173
418	369
412	13006
411	6
411	173
401	13006
391	6
391	173
392	6
392	173
383	13014
379	13
479	13
473	13017
473	13018
472	13017
472	13018
821	13021
679	13021
471	66
471	32
470	36
467	183
467	184
464	6
464	173
439	45
439	24
438	45
438	24
437	15458
511	6
511	173
523	6
523	173
522	13006
527	6
527	173
530	45
530	24
428	45
428	24
435	6
435	15472
435	1964
513	71
501	71
502	45
502	24
498	71
496	6
496	173
492	45
492	24
488	45
488	24
487	45
487	24
484	45
484	24
571	64
571	262
563	6
485	1794
485	65
568	6
568	15472
568	1964
569	355
547	36
546	36
542	65
539	6
539	173
543	15
543	6
534	6
534	173
638	6
638	173
637	45
637	24
633	45
633	24
632	45
632	24
629	45
629	24
628	262
620	65
618	6
618	173
616	6
616	173
606	71
602	32
602	15532
434	45
515	64
515	262
486	71
747	173
747	637
664	15540
721	15540
817	15540
600	15540
644	15540
877	15540
891	15540
753	15540
828	15540
751	13
790	15558
816	1
816	19
824	1
824	19
868	1
868	19
911	1
911	19
913	1
913	19
850	1
850	19
925	1
925	19
929	1
929	19
935	1
935	19
750	1
750	19
862	1
862	19
871	1
871	19
872	1
872	19
873	1
873	19
879	1
879	19
867	1
867	19
856	71
834	1
834	122
892	1
892	122
918	1
918	122
799	71
406	45
397	15845
397	172
396	71
457	369
481	6
581	6
598	6
599	6
677	6
715	6
84	1948
83	1948
622	6
622	15
626	6
626	1964
626	15472
624	6
624	1964
624	15472
690	6
690	32
678	6
678	32
673	6
673	173
731	6
731	173
688	6
688	173
589	6
589	173
557	6
557	173
590	6
590	173
683	6
683	173
739	6
739	1964
739	15472
658	6
658	240
658	15845
665	6
665	173
655	6
655	173
725	6
725	173
651	6
651	173
695	6
695	173
714	6
714	1964
714	15472
716	6
716	173
813	6
813	173
811	6
811	173
792	6
792	1964
792	15472
788	6
788	1964
788	15472
804	6
804	173
795	6
795	173
779	6
779	173
768	6
768	1964
768	15472
754	6
754	173
746	6
746	173
882	6
882	173
878	6
878	173
876	6
876	173
870	6
870	173
861	6
861	173
833	6
833	173
940	6
940	1964
940	15472
931	6
931	1964
931	15472
905	6
905	1964
905	15472
181	6
181	173
335	6
335	171
335	1948
335	173
340	6
340	171
340	1948
340	173
342	6
342	171
342	1948
342	173
326	6
326	171
326	1948
326	173
315	6
315	171
315	1948
315	173
724	13018
842	13018
843	13018
845	13018
933	13018
839	13018
839	15
724	16046
839	16046
842	16046
843	16046
845	16046
933	16046
888	64
888	262
893	64
893	262
908	64
908	262
932	64
932	262
103	262
196	262
182	262
271	64
271	262
218	262
219	262
291	64
291	262
47	262
204	64
204	262
320	262
396	64
396	262
381	64
381	262
404	64
404	262
409	262
405	64
405	262
436	262
498	64
498	262
485	64
485	262
486	64
486	262
501	64
501	262
538	64
538	262
513	64
513	262
569	64
569	262
572	64
572	262
565	64
565	262
542	64
542	262
555	64
555	262
573	64
573	262
552	64
552	262
579	64
579	262
603	64
603	262
609	64
609	262
584	64
584	262
593	64
593	262
605	64
605	262
607	64
607	262
620	64
620	262
628	64
653	64
653	262
647	64
647	262
645	64
645	262
660	64
660	262
735	64
735	262
702	64
702	262
703	64
703	262
706	64
706	262
734	64
734	262
740	64
740	262
771	64
771	262
737	64
737	262
756	64
756	262
782	64
782	262
781	64
781	262
814	64
814	262
820	64
820	262
815	64
815	262
874	64
874	262
885	64
885	262
910	64
910	262
915	64
915	262
926	64
926	262
928	64
928	262
944	64
944	262
822	64
822	262
541	64
541	262
763	64
763	262
273	6
273	1964
273	15472
156	6
156	1964
156	15472
153	6
153	1964
153	15472
114	6
114	1964
114	15472
113	6
113	1964
113	15472
65	6
65	1964
65	15472
422	6
422	1964
422	15472
480	6
480	1964
480	15472
475	6
475	1964
475	15472
652	6
652	1964
652	15472
575	24
575	45
761	227
880	227
900	45
900	16289
895	24
895	45
936	24
936	45
900	24
434	24
772	24
772	45
776	24
776	45
769	24
769	45
919	24
919	45
921	24
921	45
937	24
937	45
894	24
894	45
896	13006
187	13
187	66
221	13
221	66
332	13
332	66
406	24
532	227
532	45
604	227
604	45
806	64
883	66
883	32
884	32
884	66
659	65
798	71
791	13006
786	71
755	32
858	13006
147	13006
146	13006
840	13006
847	13006
938	13006
923	16372
923	15
923	16374
783	16393
697	1863
696	1863
705	1863
292	16397
738	16397
724	16397
809	16397
839	16397
810	16397
943	16397
553	1948
825	16405
825	16406
823	329
819	16405
819	16406
917	32
898	71
946	1
946	19
959	6
975	6
948	64
948	65
966	14
966	15
966	62
966	63
981	14
981	15
981	16
981	17
978	14
978	15
978	30
978	31
969	15
969	59
969	179
949	24
949	45
950	24
950	45
951	24
951	45
957	24
957	45
958	24
958	45
947	64
947	1283
961	32
221	1
221	19
187	1
187	19
332	1
332	19
984	70
972	64
972	262
953	87
953	88
954	87
954	88
955	87
955	88
965	14
965	15
965	39
965	40
967	71
967	72
977	13021
952	75
948	262
976	64
976	262
962	33
962	34
962	35
971	245
988	369
986	36
986	369
983	369
983	18362
987	13006
987	18362
985	6
985	173
982	18362
982	32
980	18369
980	65
979	13006
979	18362
974	18373
970	24
970	18375
968	18369
964	1969
963	18369
963	235
956	369
960	32
973	187
973	204
973	173
\.


--
-- Data for Name: transactions; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.transactions (id, statement_id, tx_date, label, amount, hash, domain_id, owner_id, display_label, merchant, label_kind) FROM stdin;
39	1	2026-01-17	De M. Madi Toumani Ben-moh	100.00	ebd5d66d63a3746c71cf2b79a3f3236fe0622892f7f119d3df48be51ddfb7917	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
13	1	2026-01-02	Carte X4519 Deliveroo 75009 Pari 01/01	-27.04	3b7786a7dd6440375361441ae423f64c17e24d5ec6fe62cf0e7bedbe562465ad	24	1	Deliveroo	Deliveroo	
12	1	2026-01-02	Carte X4519 Deliveroo 75009 Pari 01/01	-27.44	371b8f2c52504d0c78c761b9f44dd2f00f5e23a1b28e157eccc90e9d1e8435bc	24	1	Deliveroo	Deliveroo	
43	1	2026-01-19	Prlv Engie Mandat Core	-61.80	afc28de00503614adb62be9eaa4260654ec93bd40d0b43000dac87117f58cff4	22	1	Engie	Engie	
8	1	2025-12-29	Carte X4519 Ubr* Pending.uber.co 27/12	-20.91	3a45063f750b733a8a9349c8803219cadacef3a183b093562b5c9aaa5b52083e	23	1	Uber	Uber	
14	1	2026-01-02	Carte X4519 Service Navigo Paris 01/01	-22.70	0be9297c2e582afb7b08a304a6c6f6e5049509965679ef0f32d52a502a635488	23	1	Service Navigo Paris	Service Navigo Paris	
33	1	2026-01-13	Carte X4519 Amazon Payments Payl 10/01	-99.89	6f6b2835b457479dd264124446361ee48b3524bff972f281f0a3e1e63497ac56	26	1	Amazon	Amazon	
32	1	2026-01-12	Carte X4519 URSSAF Marseille 09/01	-139.00	2fe82768eebdf4e42a0f79b753c3149ee4409e6a6baefeb34732d7c748f209ee	28	1	Cotisation URSSAF	URSSAF	tax
11	1	2026-01-01	Virement Mens.pel 44690429796 01/26	-50.00	5f7975a01a7f7a9bd47a08450e62b8c26ce22cd54723b349b77d4d92cdd4e9fa	27	1	Versement PEL	PEL	savings
22	1	2026-01-06	X4519 Les Uli 05/01	-46.92	a9a1c7dd1876661284e1b3c41208d4ee3783fa58e9e0cfa0c88c47fa1131b76f	24	1	Carrefour les Ulis	Les Uli	
100	2	2026-02-17	Carte X4519 Les Uli 16/02	-36.04	0b5b61fd0e052f5f66edeb98331bbcfc62ea68db5c40b800f3a74b0450272588	24	1	Carrefour les Ulis	Les Uli	
88	2	2026-02-09	Prlv Offre Essentiel	-6.00	825a19f5bf17b33ce6041c726a4b0b35ca11e420c3a111808f5c9d8b037a09ca	30	1	Frais Bancaire	Offre Essentiel	
61	2	2026-01-26	X4519 Snbru Air Roissy 23/01	-150.00	7fec3c701955d369d334079f4077318142b64b2ace541229175ece9cb5ee0138	23	1	Frais Brussels Airline	Snbru Air Roissy	
58	2	2026-01-25	De M. Madi Toumani Ben-moh	330.00	f4c3393bc83a63ca55f9d427599d1282379da9073637f677006b9259e298a124	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
4	1	2025-12-24	Virement Lessonsharing MTB-2025-12-22F:1-02-10-1	310.00	cb52507f4a2ccafcf53f28c98e085af9f1a5428fb140773b3004489ded9f6707	28	1	LessonSharing	Lessonsharing - - -1	
3	1	2025-12-24	Virement Lessonsharing MTB-2025-12-22F:1-03-10-1	620.00	0f66c9bd92497ef6ca47dde725db95f2031bb9ee77ef1e344bc61d69a36d4bba	28	1	LessonSharing	Lessonsharing - - -1	
48	2	2026-01-22	Carte X4519 African Fashion F Pa 21/01	-47.40	1a649321ba2c305ddb24bc90efe25118b7b6fa112dd7900de65d4f6c71aa53de	33	1	African Fashion	African Fashion F Pa	
52	2	2026-01-22	X4519 Les Uli 21/01	-28.13	ab06f0e6306074ad31953ca2bd1435f4a7bec9bf2806d367067e2b5cd7510162	24	1	Carrefour les Ulis	Les Uli	
26	1	2026-01-09	Prlv Offre Essentiel	-6.00	5b7ef440df283bef8e9f1256e0650c467c6e24b360aea1e6abb4f82f54500823	30	1	Frais Bancaire	Offre Essentiel	
49	2	2026-01-22	Carte X4519 Africa Exotique Les 21/01	-11.84	a1f78f7b464f38f58792b276b53253e543730a5df0b1cc19ab0e48720a3a715b	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
55	2	2026-01-23	Virement Paypal Europe $S.a-r.l. et Cie S. YYW1047781999815YYW1047781999815/PAYPAL	162.50	f3292029bd4930e65c795807c586f7ef3d0c26dadcc2c25d991532317cba0224	26	1	Paypal	PayPal	
84	2	2026-02-04	Carte X4519 Royal Hotel Monrovia 26/01	-21.10	65e17f8c6e90fdd97d1a32fec2abe92d537e62e7f58be8390f19002176e71e6f	29	1	Royal Hotel Monrovia	Royal Hotel Monrovia	
51	2	2026-01-22	Carte X4519 Heetch”* Transport Ba 21/01	-16.00	4b45959177ac136976a171663bdee64c3fe25d295e1ca81356a477e6024518ba	23	1	Heetch	Heetch” Transport Ba	
59	2	2026-01-26	Carte X4519 Caresquick Topaze 24/01	-15.90	691f8b2c1f7db96869892766ab140d21efc48be27f83974ce2d22b86d1ebc8d2	25	1	Caresquick Topaze	Caresquick Topaze	
68	2	2026-02-01	Virement Mens.pel 44690429796 02/26	-50.00	d244ab61968f87a48f10913e1e73315dd31b4c0d75b8f28bb00ccad2a11b762a	27	1	Versement PEL	PEL	savings
140	3	2026-03-13	X4519 Les Uli 12/03	-51.40	99a98960ba1a900e5104cb0f792306684bb5e434ee88087b251455c74b046922	24	1	Carrefour les Ulis	Les Uli	
86	2	2026-02-06	Prlv Acheel France REF-1CBBB9880770F24695GFGWBH4 FR457Z787F911 Core	-12.47	961ea78d18210e8de49469e821468b9595cf48e8a528928178a3261baf433470	22	1	Assurance habitation Acheel France	Acheel France Core	
34	1	2026-01-13	Carte X4519 Nyx*justeatemps Rois 13/01	-0.50	f7c1815c919710fccfd8741568ef72151d3c9fd0c9f5e9a0836c16de6e1b4ec2	28	1	Machine à café	Nyx Justeatemps Rois	
139	3	2026-03-13	X4519 Africa Exotique Les 12/03	-29.50	ca5ec60b93f64bbc37538314e3963ec597e41ea0ef297bdef2a2ed3ff4e9b617	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
116	3	2026-02-25	De M. Madi Toumani Ben-moh	200.00	baf171382c3abeaf99b3f5c4209cc53bd2191c3989ca5e965a8999eb6548fb28	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
27	1	2026-01-09	X4519 Nyx*stcommerce Paris 08/01	-0.70	0827f75dbe73f60642fc5e37ac0a9dfdbdc43a683b930f7001de367d863c88da	28	1	Machine à café Paris	Nyx Stcommerce Paris	
142	3	2026-03-14	De M. Madi Toumani Ben-moh	50.00	ce01063b1aa242e5b5634a8a7b30aed9b7f2374afd509bc4b8097b8d6be20889	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
143	3	2026-03-14	Virement De M. Madi Toumani Ben-moh	100.00	d8646b3acb21a81bc0979155ba04b0403c7d430ce3a17e105c71c2d02bff9c34	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
90	2	2026-02-10	Carte X4519 Ornikar 75019 Paris 09/02	-7.99	3e218a7d7daf422088ca1c2f530a29f61813dc0398ab0a2ec2c092d188342b2f	29	1	Ornikar Paris	Ornikar Paris	
129	3	2026-03-06	Carte X4519 Deliveroo 75009 Pari 05/03	-22.39	85fff82b578be5c1fc12d9e49d84f703fecac5ab39184f96854f5a875fe7c7be	24	1	Deliveroo	Deliveroo	
10	1	2026-01-01	Virement Vir Inst de Mile Aya N’goran	300.00	7720b564d9de37cbcc2a7d3ca48625bbcba91d21f51e04130cb568719038c346	33	1	De Sonia	De Mile Aya N’goran	
30	1	2026-01-10	Vir Inst de M. Madi Tarik	50.00	fc7576b95382909d912f421f66826cdfd96980265b42772e0153a50b13a45f94	29	1	De Tarik	De M. Madi Tarik	
46	1	2026-01-21	De M. Madi Toumani Ben-moh	400.00	1c194cbab45ee6c151145ca666ab1565f46b248da6692d5cb1aba2ac14bee4ff	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
85	2	2026-02-05	Prlv Free Telecom Free Hautdebit 144933115747939382 FR83277459654 Core	-29.99	4436b61f7817dcf807ce60e931fbcb0ab19db8804a7719c886874175741c5432	22	1	Free	Free	
110	2	2026-02-20	De M. Madi Toumani Ben-moh	980.00	4da68e0171bc00a8d5960b7041df4f6cc0f6669ad6ba958df5e475396330e5e8	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
62	2	2026-01-26	X4519 Ubr* Pending.uber.co 23/01	-52.99	5b981d13b69d722f7f039e71d60b7433d763c2c57b82e2eca26a407eaa586eb3	23	1	Uber	Uber	
102	2	2026-02-17	Carte X4519 Service Navigo Paris 16/02	-2.55	861c2db48f41609a8cc1f44a5cbb2f0e9b426a2879979619d215127335f8e4ab	23	1	Service Navigo Paris	Service Navigo Paris	
96	2	2026-02-16	Carte X4519 Amazon Payments Payl 09/02	-36.18	52c3786c980bc1189b0325ebad36e205bf5fd15095aa62a23f41835b483109fb	26	1	Amazon	Amazon	
147	3	2026-03-16	Carte X4519 Sncf-voyageurs Paris 14/03	-39.00	85117a0fc06d7af1be2a930b9049a72e029b267e88674de24feb6ae38893d0b8	23	1	SNCF	SNCF	
119	3	2026-02-26	Carte X4519 Sloann Paris 12 25/02	-19.50	addc7f016af52755730b095c25d6db1760387a2f5867ab9ebe5cb608eeb36003	28	1	Sloann Paris	Sloann Paris	
123	3	2026-03-01	Virement Mens.pel 44690429796 03/26	-50.00	a1d862e5b9380954fd2bc0b6e12ebdbc18ab4aa4838ebe499720450c47155fd8	27	1	Versement PEL	PEL	savings
120	3	2026-02-26	Carte X4519 Ubr”* Pending.uber.co 25/02	-7.20	8c39e68f2757eb10f8deaff2b59d6e2deb933378868905152d8ea590b2083bbb	23	1	Uber	Uber	
171	4	2026-03-31	X4519 Les Uli 30/03	-34.53	f8ae39c3506ad392facbd569d9785e56fb0c099e6c737e0b29b5fd8ad7650d22	24	1	Carrefour les Ulis	Les Uli	
121	3	2026-02-27	Carte X4519 Ubr”* Pending.uber.co 25/02	-28.91	60f718be85b3aa0801d359a34a625858f4b9dce1608d3f117a749d5f7e77844d	23	1	Uber	Uber	
195	4	2026-04-13	X4519 Les Uli 10/04	-61.72	e5d935cd879929b4ae1341b4b970362aded9d198eb6073190d16ed07fbb465f6	24	1	Carrefour les Ulis	Les Uli	
190	4	2026-04-09	Prlv Offre Essentiel	-6.00	9be3661107ecfd1df35ac4fe69feb26ed3cf9d560bb6d14266080e6ba77d67c3	30	1	Frais Bancaire	Offre Essentiel	
194	4	2026-04-13	X4519 Africa Exotique Les 10/04	-10.55	1b77baaf2e5e4461150b243a00dc1b04fecd22083984c8f28019c95477524ba9	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
166	4	2026-03-26	Vir Inst vers Madi Toumani Sakin	-650.00	1464b57414435a94a03563ae446ea3241f62d859bc8e4e2b29f192c9c9af43b7	29	1	Vers Sakina	Vers Madi Toumani Sakin	
196	4	2026-04-13	Carte X4519 Ulis Viandes Les Uli 10/04	-39.27	c9d4a9fd281784096a2737f0c620d346b92fd771ef20219bb2ea92683af6f1aa	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
180	4	2026-04-03	Virement Vir Inst vers Mme Jing An	-2850.00	ef09e4d4989cd0945928d5e44bfa98372b5c41715e22fcdfc42f2512f1b03f99	22	1	Loyer	Vers Mme Jing An	
151	3	2026-03-19	De M. Madi Toumani Ben-moh	250.00	cbfae6942e2c7455570ccb905055c69f3d912247367938cbbb7fcaefd6644078	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
155	4	2026-03-22	De M. Madi Toumani Ben-moh	60.00	36c672fe934edc2c1f0a89d195357821736c28f4a0a670c77b3ac81905db723e	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
165	4	2026-03-26	De M. Madi Toumani Ben-moh	650.00	29170ff6fe6167691e57855fcf2440b190591d2030541bdceeaa84ba7517d659	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
168	4	2026-03-28	De M. Madi Toumani Ben-moh	200.00	5b7e382b0c9bd4e792c71b169a46b540aef75812faf9a3474b84da8188dcaddc	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
133	3	2026-03-09	Prlv Offre Essentiel	-6.00	a5010453db428530a2b8a8cf6832923d496b0ae6930687af12a21f6541f37b83	30	1	Frais Bancaire	Offre Essentiel	
199	4	2026-04-15	De M. Madi Toumani Ben-moh	250.00	149c037a5366e7863e56fc629b27d39e53db67fb026efb95fff827728a8c619e	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
158	4	2026-03-22	Web M. Madi Toumani Ben-moha	-30.00	72a099afa3cff15782a4059fb011f7e47f5b65167d1647e883094ef6ae6acb21	31	1	Vers Livret A	Web M. Madi Toumani Ben-moha	
188	4	2026-04-08	Prlv Acheel France Core	-12.47	f486bf3def29010439235c3a5852185cbb9956891e05670f9d0d299c2e131a36	22	1	Assurance habitation Acheel France	Acheel France Core	
189	4	2026-04-08	Prlv Free Telecom Free Hautdebit 146452883347939382 FR83277459654 Core	-29.99	195ed1953779b9c8059c289dcba69a2d336b5af470a0d29cf8532b1fdb002d60	22	1	Free	Free	
136	3	2026-03-10	De M. Madi Toumani Ben-moh	200.00	cbe6ec5be5b613d9b4b2e651b6ca6274ea7eaaf56ec909fe495cff306d50a718	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
167	4	2026-03-27	Virement Vir Inst de Paypal	162.50	34bae17528cb373f46d5626a9438953b5fdf6881cec07fbeb9869f69e4a8cb8c	31	1	Paypal	PayPal	
176	4	2026-04-02	Carte X4519 Amazon Payments Payl 31/03	-39.11	938299285acc9037c3c1cd8438a6be37118f0f0f8d2975bc1b544a167f26c29c	26	1	Amazon	Amazon	
184	4	2026-04-07	Carte X4519 Amazon Payments Payl 02/04	-38.98	b40154a1320b13c835e64dde2531d8befd5680349fe6b349b16ff1a006862aa9	26	1	Amazon	Amazon	
183	4	2026-04-07	Carte X4519 Amazon Eu SARL Payli 02/04	-18.59	0754c2dd8f3a0c0d38624aaedec5ff49275adc3fb668a135ee58097a33e3a79b	26	1	Amazon	Amazon	
182	4	2026-04-07	Carte X4519 Action Les Ulis 03/04	-9.00	78db49e20bce828f9e09f217e0f95669ecf99fd3e84b6cec4ddcb28b692a0383	24	1	Action	Action	
175	4	2026-04-01	Mens.pel 44690429796 04/26	-50.00	1ef6a98f48fcad7e531ff59c6ccf219870db77ab9368e3a83e23b8f6efe59c7e	27	1	Versement PEL	PEL	savings
174	4	2026-04-01	Virement Lessonsharing MTB-2026-03-31F:02-02-1	175.00	baac190d8fb9411472ae2778588fcbda61e079bf1205d590ce2e0b855b18baf8	28	1	LessonSharing	Lessonsharing - - -1	
101	2	2026-02-17	X4519 Pizza Andiamo Les UI 16/02	-9.00	fc86af56947c4ef23ed97e32ad200e87cd45dfb60808802400400b6bcc0339c7	24	1	Andiamo Pizza	Pizza Andiamo Les Ui	
19	1	2026-01-05	Vir Inst vers Ben-mohamed Madi T	-80.00	daa4ed45d20d5a4f5946c06563c03c280bec6044dfc50ad05e8b92f361d01e43	24	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
60	2	2026-01-26	Carte X4519 Snbru AIR08223037157 23/01	-187.65	79f46248f53fe8c21e355bb96f7422a80f1b60d9f57c50db54a492758ee7c9d3	23	1	SNBRU	Snbru	
53	2	2026-01-22	Carte X4519 Phcie Courdimanc Les 21/01	-19.80	18098506adea77bc947fd60843cea1ea5347b41e9637e7789ba2e38012c8d7ec	25	1	Phcie Courdimanc	Phcie Courdimanc Les	
108	2	2026-02-19	Virement De M. Madi Toumani Ben-moh	750.00	594c20e97ff9c1649fc2bc26110df780f414cd967bc6b90b52019bcad4052cf4	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
209	5	2026-04-25	De M. Madi Toumani Ben-moh	150.00	5b37ec7de08a6b8f1215df1ae3804ce9bec0b53bfcf82bd8c2c52b32dc7b16c4	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
225	5	2026-05-13	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-11.99	a648ef643c35c99f950273b9ef1105b9eb99085ca6f965fbd7ef78d7e5732d0c	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
222	5	2026-05-06	Prlv Free Telecom Free Hautdebit 147098249247939382 Core	-29.99	96fa685944142c298f92d1c47724dd582f40f0310e351959727f8162eebb5cc7	22	1	Free	Free	
227	5	2026-05-19	Prlv Engie Mandat Core	-54.88	ab72a42c561000ebb0d822c9dd9f1adc45fa05ef439e9c8c0cefc1dde291c4b3	22	1	Engie	Engie	
274	7	2026-05-26	Carte X4519 Mp*carrefour Les Uli 23/05	-21.31	862bcdb2820dc1869dffff6925ffa6ad9a9c51f5b396329b8ba7dda7b9c11a6c	24	1	Carrefour	Carrefour	
275	7	2026-05-29	Carte X4519 Mp*carrefour Les Uli 28/05	-19.00	36320f4a458e5cb09c466baf3708b5a84225e26e74e538107fa4d9d7d43835a6	24	1	Carrefour	Carrefour	
271	7	2026-05-22	Carte X4519 Burger King Les Ulis 21/05	-23.10	00fb7390e45003c329f9c24277160d23419b182955aad6ab613362e4552126a5	24	1	Burger King	Burger King	
210	5	2026-04-27	Avoir Carte X4519 Amazon Payment 24/04	24.99	f49143e8484104de6766470b741f4f69323564f3600fc039da919d102b24d2c8	30	1	Amazon	Amazon	
203	4	2026-04-20	Avoir Carte X4519 Amazon Payment 18/04	27.98	97568e704d03a1d4e15007cc67de513f4a520d250120d65d9583eae6e5bf482e	30	1	Amazon	Amazon	
212	5	2026-04-28	Carte X4519 Five Pizza Les UI 27/04	-17.10	7f4e2312ab3a87c9d61b26578f9414d003161147900ab941ff325e878dc95f17	26	1	Five Pizza Les Uis	Five Pizza Les Ui	
215	5	2026-05-01	Virement Mens.pel 44690429796 05/26	-50.00	e5563379d0c7c8128be15a3b05ac8ce0dd39e1268332502548faf18b8802d37f	27	1	Versement PEL	PEL	savings
157	4	2026-03-22	Wero vers Aleksandar S	-140.00	ae9597f0d4f66fcdf4740ec0f628dc275af84cd758aeaabbe57ca0e979dee566	29	1	Wero Aleksandar	Wero Vers Aleksandar S	
218	5	2026-05-04	Carte X4519 H.t. Food Les Ulis 30/04	-8.80	d829463c6f73c499b601dfeeb19904fa87d13c70bc767e514dfaff249d302e9a	24	1	Les Ulis Food	Food Les Ulis	
276	7	2026-06-01	Virement Mens.pel 44690429796 06/26	-50.00	3d68970445c11ef896dc560be7264be696ae059d59aa2b30fffe99c01251cf59	27	1	Versement PEL	PEL	savings
303	7	2026-06-11	X4519 Les Uli 10/06	-39.46	f183b924f495a4bcb61f1290efe94b45d484350c20f97f6209594b94f8d82205	24	1	Carrefour les Ulis	Les Uli	
192	4	2026-04-11	De M. Madi Toumani Ben-moh	50.00	02f6177edfbc886eb86432f9253c677be4332ca1d10f256f0e72d2d22f1e0a56	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
18	1	2026-01-05	Virement Vir Inst vers Mme Jing An	-2850.00	b45c1aa3cc1cef11a2a8cbc831d0a96b679f2b4f77b7bd52772a1fb99b57d89e	22	1	Loyer	Vers Mme Jing An	
292	7	2026-06-08	Carte X4519 Shein Dublin 07/06	-39.43	5522969788a2ebb991c199040bab614650d245144b7da02f565090deaffcb08c	29	1	Shein	Shein Dublin	
16	1	2026-01-04	Virement De M. Madi Toumani Ben-moh	800.00	0e1767051c00591eee3b6c0d024607d981ebbc4bfae4065fa9d1826fb7df5573	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
42	1	2026-01-19	Virement De M. Madi Toumani Ben-moh	200.00	7e55d7d9dfb3dd5519af040e3351af479e713e93d031008025274e21c0ada58f	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
54	2	2026-01-23	Virement De M. Madi Toumani Ben-moh	200.00	9701cc97b070cf50262972fb7332c5d49b15f8fc5a814e91ac747e55782f868c	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
299	7	2026-06-10	De M. Madi Toumani Ben-moh	200.00	f9c833af81dbcf6a5d7d729d805cef36e6a29acbc4beec902de0d2263db6a9e0	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
160	4	2026-03-23	Carte X4519 Deliveroo 75009 Pari 22/03	-25.24	0c44643271857b8eaaf3a935f717deafc637bb9b2629ec8ef568ec1444bfe5cc	24	1	Deliveroo	Deliveroo	
6	1	2025-12-29	Carte X4519 Hippo Bercy Paris 27/12	-170.80	39acd6b4d304231c5bc92aa9a018c7579a88c3bc3a0df9d9fc6cec8f3547fa92	33	1	Hippo Bercy	Hippo Bercy Paris	
205	4	2026-04-20	Ret DAB X4519 Les Ulis 18/04 14H30	-20.00	03eedebe422481fc80848603f707126bca5baa6d1f5e7d8cf6e7e1adc460482a	30	1	Retrait Distributeur Les Ulis	Ret Les Ulis	
226	5	2026-05-18	X4519 Les Uli 15/05	-39.43	8bc39cd10aab4d283f6de1ac87ee5072db9627643418ac5e95ba220fe3794c1d	24	1	Carrefour les Ulis	Les Uli	
223	5	2026-05-09	Prlv Offre Essentiel	-6.00	c616d15df65cdb799f20d4d1caed18357eb7dd1bf43e42b64a7a9986f255af55	30	1	Frais Bancaire	Offre Essentiel	
221	5	2026-05-06	Vir Inst de Madi Toumani Sakina	300.00	83e15c79135019cd3fcc705c3a3f5d3a72b6fbe8c19b9a7dd19fec3f835fea34	29	1	De Sakina	De Madi Toumani Sakina	
285	7	2026-06-05	Prlv Free Telecom Free Hautdebit 147851605447939382 Core	-29.99	d2a231177c78374caf88094d211174d4f1756b94826d75d1ce3145e9a113c1bc	22	1	Free	Free	
310	7	2026-06-17	Prlv Engie Mandat Core	-54.88	216ee2d4ee4a4d667c3064761a647b0c680345f868ad2eff31565f153567e017	22	1	Engie	Engie	
291	7	2026-06-08	Carte X4519 Burger King Les Ulis 05/06	-12.00	0e6115b62df271058f121355a4557ff637e97c00a678999597738eb73619be9b	24	1	Burger King	Burger King	
304	7	2026-06-12	Vir Inst de Paypal	197.79	9d3967e0dd229d9cf46629c23f485c27321e09a00c5cbc649c7972362876c3cc	30	1	Paypal	PayPal	
312	7	2026-06-22	Carte X4519 Cofidis Amazon Ville 20/06	-63.68	f5fcfdbb5cf97ecf5af1830673a6c973d6159348ad10c0c33868658d1f22b69c	29	1	Amazon	Amazon	
300	7	2026-06-10	Carte X4519 Amazon Payments Pari 08/06	-186.61	21729ff974875318fbec55a21b33df5784f33550a2f7c1c2c797bfada050c38c	29	1	Amazon	Amazon	
307	7	2026-06-14	Vir Inst vers Assjad Saida	-105.00	49065f520c12d399470cfd97dea31427c9d48662c6b8deef1b0a8ad4e91bc3eb	29	1	Vers Assjad	Vers Assjad Saida	
47	2	2026-01-22	Carte X4519 Action Les Ulis 21/01	-11.83	2187d32bc2308795449d84a938a571080c4a4e91b5ce512e5f00735afc27c1b6	24	1	Action	Action	
146	3	2026-03-16	X4519 Lw*sncf Connect Pari 13/03	-41.10	95d06c24f6745a664266e2888bc882af62c721ad763931348248ee85d9366c1f	23	1	SNCF	SNCF	
277	7	2026-06-03	Virement Vir Inst de M. Madi Tarik	300.00	12c51872f348e194c4da397b1fcd41d6d3e018b1f6dc23a2784fdf9096551c9f	29	1	De Tarik	De M. Madi Tarik	
202	4	2026-04-18	De M. Madi Toumani Ben-moh	40.00	506fcc98154e682545e79bb47e13a832ef2826880cb7e58302a75cdb19e6318a	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
99	2	2026-02-17	Carte X4519 Africa Exotique Les 16/02	-10.96	3f3aa668f031ca5396049865bbcadf0fde458cd109e48d36dd9258f4861650f9	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
80	2	2026-02-03	Vir Inst vers Madi Toumani Sakin	-100.00	aca3adde8ef70a5231b8a0f4e290b1c5f92cff3dd12770b9c6d03a0d12e93a0a	29	1	Vers Sakina	Vers Madi Toumani Sakin	
66	2	2026-01-29	De M. Madi Toumani Ben-moh	500.00	765b07c472fe44a14a1e20e8c16f115a7e0aaa57811c8f632c3b8fb77c2e48ad	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
98	2	2026-02-17	Virement De M. Madi Toumani Ben-moh	400.00	867e075cbc1f47df32069dc47283e25252077562699be828cbea1588dbc8519a	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
148	3	2026-03-17	Virement De M. Madi Toumani Ben-moh	50.00	14e4583fc220688f3e9ad52924e7ca2467166c4c8d4272bf4ecedb337aa1405e	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
154	4	2026-03-22	De M. Madi Toumani Ben-moh	150.00	b650ab801e653e41ee7b1c33ea11f4c937e33a9d104d00082089ed39f2c3fa20	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
159	4	2026-03-23	De M. Madi Toumani Ben-moh	100.00	359ef2148957714350ae16bf62fe62f2b1c3c0caed69320ccc5f33dee45f47ae	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
82	2	2026-02-03	Web M. Madi Toumani Ben-moha	-5000.00	b94c5b597eb9e9a4e7a464552388406cd1d773bbc903b57aed58480d75a8ecb6	31	1	Vers Livret A	Web M. Madi Toumani Ben-moha	
152	3	2026-03-20	Carte X4519 Deliveroo 75009 Pari 20/03	-26.29	9144908c70d85e6842ac195c8c870a42d0edb8a000dbd5ab16a6ba05dbf097e9	24	1	Deliveroo	Deliveroo	
1	1	2025-12-24	Virement Lessonsharing MTB-2025-12-22 F:1-27-09-1	280.00	e84851419d8533a560c0ec447688c90ac20cb2a90a65f0eff33883c9d3f6059b	28	1	LessonSharing	Lessonsharing - - - -1	
298	7	2026-06-09	Ret DAB X4519 Les Ulis 06/06 09H28	-20.00	6811c5838095a6de2184ce8f845aa68d852d8ba8522449e90d118f5b0fd9df45	30	1	Retrait Distributeur Les Ulis	Ret Les Ulis	
289	7	2026-06-05	Web M. Madi Toumani Ben-moha	-2850.00	a2d9fc28a764971e73db5eaa0a5ba7ab329fa9f666b62730639f283b79b8aa1d	31	1	Vers Livret A	Web M. Madi Toumani Ben-moha	
28	1	2026-01-09	Prlv Acheel France REF-927B2080281923C168GFGWBH4 Core	-18.90	4daf86469e7f882869b03bd4cebd590d9fd08161529330929d87f3fd66b79af9	22	1	Assurance habitation Acheel France	Acheel France Core	
294	7	2026-06-08	X4519 Viandes Du 95 Argent 06/06	-7.50	82ae17ab488d668147c9f5a71a2c6eaa7dd4bdd670080b1c30ff0596deea926d	24	1	Boucherie Argenteuil	Viandes Du Argent	
198	4	2026-04-14	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-15.97	1ffe2c0cd3c623b0668bb41b1ff3f172a8ff7e58060627b82dc5889b84131799	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
306	7	2026-06-12	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-13.99	84a561d9edbc429a287ca123c1a5213c67c0b08830d0c032b313e66b36671221	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
23	1	2026-01-07	Prlv Free Telecom Free Hautdebit 144160875547939382 Core	-19.99	80da31ab0823bea7678f818fc9030042517bdea30aef68cc50b9e492530395e0	22	1	Free	Free	
279	7	2026-06-05	Virement Lessonsharing 2026-06-04-MTBF:1-19-03-1	560.00	ae141762f5c8508fd1f5f11aad8c4d642bf28cc8b4e920f45adac9e3edca6307	28	1	LessonSharing	Lessonsharing - - - -1	
83	2	2026-02-04	Carte X4519 Royal Hotel Monrovia 26/01	-46.49	25679694dd4ea7552feedf083cda6e0380e63e7fdcca87fa8e6ac11437bbbe38	29	1	Royal Hotel Monrovia	Royal Hotel Monrovia	
57	2	2026-01-24	Virement Vir Inst vers Ben-mohamed Madi T	-120.00	8bc65e8f13143d4f44556dfa17f050f386fbc6339bd586b730bfcafec02d63c3	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
64	2	2026-01-21	Virement Vir Inst vers Ben-mohamed Madi T	-280.00	53e5f3e8b5907cd3c8837e7e623a2f9b421aa79d89ecebd96ae6a606ff79211f	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
93	2	2026-02-12	Vir Inst vers Ben-mohamed Madi T	-91.00	6a17215a46b78a98a04184beb3671a9912e885230975b07e5829a25c6b57bb48	25	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
67	2	2026-01-29	Vir Inst vers Ben-mohamed Madi T	-100.00	d4ba6ad37c4209dd295b2b5428f2116c382a1ce1d2352eb54ad35b70e7bef087	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
79	2	2026-02-03	Vir Inst vers Ben-mohamed Madi T	-450.00	2d14bb2ed0895d6d0e90cb5c471d183e3ec1b17eb02000e9b3670ba0922a9647	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
91	2	2026-02-10	Vir Inst vers Ben-mohamed Madi T	-180.00	e9bb5c4389204e37963eb2114a61a65e049998c16918ed42a90b04692e179713	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
107	2	2026-02-18	Vir Inst vers Ben-mohamed Madi T	-30.00	bffffc774f1193185d93dc3a1d48454bcd6b00558fa6177437b7e556096d90f8	30	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
113	2	2026-02-20	Vir Inst vers Ben-mohamed Madi T	-30.00	2dc849fa30075af0d70dd46bbe21f9c22eeeaa93e4a6c8aaf27bba7da80d7288	44	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
284	7	2026-06-05	Virement Lessonsharing 2026-06-04-MTB 00004F:1-07-04-1	1210.00	f6294c2114c8fe65343a699c4f22578c22d7c7feee99b83e06472293c67193ca	28	1	LessonSharing	Lessonsharing - - - -1	
153	3	2026-03-20	Vir Inst vers Ben-mohamed Madi T	-30.00	13f6869849f9e4e67a44789568562a78e53694843579d49fd69e3dcbe39d7d6f	44	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
122	3	2026-02-28	Vir Inst vers Ben-mohamed Madi T	-20.00	11da7999aa4bbabce47de6aa9e3b24af3a3962d1d8533a706e3e7c67134b6af8	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
125	3	2026-03-02	Vir Inst vers Ben-mohamed Madi T	-250.00	8893ba636aad2850fd296545a7682bcbee6c4f6eb1e793d5087ef2319118daf5	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
131	3	2026-03-06	Virement Vir Inst vers Ben-mohamed Madi T	-105.00	8d4c48d240a207bb80b7cd5629f0e1bb72f8673d11385aa84770add314c4ea10	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
149	3	2026-03-17	Vir Inst vers Ben-mohamed Madi T	-49.00	99f1a3a987b7fd0bb15d274e13046eb17c7fb094f679a3db73546749c8f51d70	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
311	7	2026-06-19	Carte X4519 URSSAF Idf Montreuil 18/06	-17.00	bcf514e2889f4e19917dc63e2de8b0d8be5b71abb2800b79dce11aa2d931ef56	28	1	Cotisation URSSAF	URSSAF	tax
106	2	2026-02-18	Carte URSSAF Marseille 17/02	-754.00	27e67c554cd7e8308435ea288bf19e65222926a68a283618996188309837ce5c	28	1	Cotisation URSSAF	URSSAF	tax
191	4	2026-04-09	Carte X4519 URSSAF Idf St Denis 08/04	-361.00	5cfa8f17786f0c0f27e584c9b828c5631fc9f65c3cca096d2107dad2354d576e	28	1	Cotisation URSSAF	URSSAF	tax
287	7	2026-06-05	Vir Inst vers M.madi Tarik	-300.00	5e0ab6c1e8e4641ca0285c8de9415d38e93dfaa32daed560c87875d26b2d2e44	29	1	Vers Tarik	Vers M.madi Tarik	
290	7	2026-06-06	Virement Vir Inst vers Aya N’goran	-45.00	ce70bb00bc6d888ed4a4a778b2b6a571fc3697ef402816ce2720b1ef69e18878	33	1	Vers Sonia	Vers Aya N’goran	
173	4	2026-04-01	Virement Lessonsharing MTB-2026-03-31F:1-19-01-1	1210.00	1607f6f715f6160493ac6f64bdcf5cc83ec507b562facb30309fd2dc8f9001f0	28	1	LessonSharing	Lessonsharing - - -1	
156	4	2026-03-22	Vir Inst vers Ben-mohamed Madi T	-26.00	65174ba330f104f5aa61a8c5560abeb6a6ae85a13d78164e717a5c24d39270a9	44	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
305	7	2026-06-12	Carte X4519 Tiktok Shop Seller D 12/06	-12.38	cff73d34a39e074597deebf095e3ed9afdccafd1eeda3f64199c678a68d1dcd6	29	1	Tiktok Shop	Tiktok Shop Seller D	
201	4	2026-04-17	Prlv Engie Mandat Core	-54.88	c04021eb97430308e27b15573eef3c6f6ecccfb6d39f8c12b5161094bdf7614d	22	1	Engie	Engie	
181	4	2026-04-03	Vir Inst vers Ben-mohamed Madi T	-50.00	475d780a0c04f49d9ecb33f99bbee5d0e385e01111421b041e8123341115b632	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
186	4	2026-04-07	Vir Inst vers Ben-mohamed Madi T	-50.00	65cfa3af9a6f7104bda1ed8bb4ff693989389a197738cfd5c7394203401ae932	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
193	4	2026-04-11	Vir Inst vers Ben-mohamed Madi T	-45.00	e2aede1618600129be31184e482d6565c33e451ebc7d954b978669d11af6d682	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
216	5	2026-05-01	Vir Inst vers Ben-mohamed Madi T	-25.00	b6550cc1ffc04da5457239c79e028377d965ceef03fa2f001e3b12a352d124ef	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
45	1	2026-01-19	Vir Inst vers Ben-mohamed Madi T	-53.00	78980e08109702ffb58365039bbabf79021159b5aeb19fb2f68750390e87c83e	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
301	7	2026-06-10	Prlv Paypal (europe) $.a r.l. et Cie, 1050858593549 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-287.09	dd400ff021a7fda89d8e48f575986f2f48947b4cdd90cd19f079d43c383434d2	26	1	Paypal	PayPal	
29	1	2026-01-09	Vir Inst vers Ben-mohamed Madi T	-650.00	50455fbe38032fdec4de80a5ba5ccb02a7f7c5bb24546e47c00dc52d9d4dccb8	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
286	7	2026-06-05	Vir Inst vers Ben-mohamed Madi T	-354.00	46d6b8376a1e3edfae974cd4d80646858a47c22bea9b0e89710ece6c27baf91c	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
50	2	2026-01-22	X4519 Heetch* Transport Ba 21/01	-18.00	9da66a6190075089a8560cce0f06e36145d8c8443669d01a0d7d6169f8c21ca6	23	1	Heetch	Heetch Transport Ba	
63	2	2026-01-26	Vir Inst vers Ben-mohamed Madi T	-52.00	e728d6d200cc60f6ac34e36a001c78937162a257b3390178885ba28d36e655f9	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
89	2	2026-02-09	Vir Inst vers Ben-mohamed Madi T	-100.00	843fe0b254991120868056893249e570447b42544eb6d79aa52ddc71037d508b	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
94	2	2026-02-13	Vir Inst vers Ben-mohamed Madi T	-80.00	697ca8d4f45b52c3ab59cf0296f468a4619bb4f081b7bb16706cac87f244760b	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
109	2	2026-02-19	Vir Inst vers Ben-mohamed Madi T	-720.00	2f18eadc8faa829d7ff2e46e388c9a87b49cc1aa2bd438cadb98d46116ec2237	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
117	3	2026-02-25	Vir Inst vers Ben-mohamed Madi T	-20.00	19ed22690b98a4a802649dc9cf5c1b3a99ad4a1e502e7a657586fc64254799dd	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
135	3	2026-03-09	Vir Inst vers Ben-mohamed Madi T	-110.00	44c89b7edd495f222b46b4c1f54e775fd2d63be352438b4bcd563534579aa821	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
138	3	2026-03-11	Vir Inst vers Ben-mohamed Madi T	-50.00	9b8ec3259926834c9336672d1605c4b1f01ecc89f00ae492c97e06a1fd187147	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
144	3	2026-03-14	Vir Inst vers Ben-mohamed Madi T	-50.00	34452a9406513e50302412c8d29b6a43904d96f0f2cafe3bf1821d76485f14d3	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
169	4	2026-03-28	Vir Inst vers Ben-mohamed Madi T	-380.00	0dd1d7980c18798bd907d66017c99f63380833eb51ee2b10cf18542f28c116dd	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
309	7	2026-06-15	Virement Vir Inst vers Ben-mohamed Madi T	-605.00	e606650e849cea3b6ffee439ee5c43c40aba2be038b52d2d01e64dbd72f5ce37	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
111	2	2026-02-20	De M. Madi Toumani Ben-moh	30.00	7ad9e59b96fbaba88523ec0e572fd9fdb792f0b0dfd8a2fd383c2cbe13e2b079	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
37	1	2026-01-15	Carte X4519 Nyx*justeatemps Rois 14/01	-0.50	748e8db9fe07066a7907b8bd97e02d95cce7623f9fa714842bcc7b6185d4c26c	28	1	Machine à café	Nyx Justeatemps Rois	
105	2	2026-02-18	Carte X4519 URSSAF Marseille 17/02	-2291.00	ee552ebcb319614f73929a7aeafc93e3b39219088d3cd70729d37c8494fd2870	28	1	Cotisation URSSAF	URSSAF	tax
141	3	2026-03-13	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-11.99	2850fa332943a70afc91e547a9ecb92664fdd8867b5eb9e1006d896bd81acd07	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
187	4	2026-04-08	Virement Vir Inst de Madi Toumani Sakina	650.00	ca8399e295c0b62d9f4e480f590f16e129a10ac4f177ce073401f0f5dff73ffa	29	1	De Sakina	De Madi Toumani Sakina	
213	5	2026-04-29	Virement Vir Inst de Paypal	241.25	abbd4c416140574211add26743213391010e1508091719e7b72d7f717385f7d6	31	1	Paypal	PayPal	
217	5	2026-05-02	Virement Vir Inst de Paypal	162.50	d8c0af2a6ca12cc7f471aa89f461550113e4aa01f186e91ccfba75cb60d85336	31	1	Paypal	PayPal	
214	5	2026-04-30	Prlv Paypal (europe) $.a r.l. et Cie, 1049930192044 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-195.95	b042a44e79c8a47cd911f13cecef5392890bc295f9a56c9ab2929fbf42deee13	26	1	Paypal	PayPal	
56	2	2026-01-23	Prlv Paypal (europe) $.a r.l. et Cie, 1047754968077 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-162.50	e16160ad3d5825b75ceecb0471310a327577a69e5a30a47f19475f35f3cde1b8	26	1	Paypal	PayPal	
150	3	2026-03-18	Prlv Engie Mandat Core	-54.88	a77146fb92bae905d5acb42652eb6b7409a0b44c084c8725cad28afb44abc05e	22	1	Engie	Engie	
162	4	2026-03-23	Vir Inst vers Ben-mohamed Madi T	-30.00	065dc62f4a7c4ce8467adcbef45ff82e6b65c705a35c00949ca51fa96ecfeb94	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
172	4	2026-03-31	Vir Inst vers Ben-mohamed Madi T	-50.00	5078a39526a3b4f216ce9c81f9ec25f3a773ca12ab17758709562f2e032bca8f	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
228	5	2026-05-20	Carte X4519 Cofidis Amazon Ville 07/05	-63.68	907bde8a4099efe42e8924859bda8ecaebad1a92b7b1e3efe8d46bf1b19cedc4	29	1	Amazon	Amazon	
178	4	2026-04-02	Vir Inst vers Ben-mohamed Madi T	-50.00	f2c60d1ea369d5662d30d3798c43c9620f499fe486893e2219b61850656c2d7e	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
76	2	2026-02-03	Lessonsharing MTB-2026-01-30F:1-22-11-1	620.00	8cbd785650c0d74107e36d10af4611c88c095e309388a6f2dc2518072e077a02	28	1	LessonSharing	Lessonsharing - - -1	
74	2	2026-02-03	Virement Lessonsharing MTB-2026-01-30F:1-02-12-1	100.00	8485edce56725fdca0c4c4c4f460d158f08006fc21e3294140d63bc2f17499c4	28	1	LessonSharing	Lessonsharing - - -1	
124	3	2026-03-02	Virement Lessonsharing MTB-2026-02-27F:1-12-01-1	1400.00	c55b1538dbdb7159ea09fe19f4657121300438659a24a81b0175f70736886834	28	1	LessonSharing	Lessonsharing - - -1	
69	2	2026-02-03	Virement Lessonsharing MTB-2026-01-30F:1-12-12-1	1120.00	3bf7ad0d49aa12afb2631cbb312649f45c638fb3b78bbef273c0decc88188c5c	28	1	LessonSharing	Lessonsharing - - -1	
9	1	2025-12-29	Virement Vir Inst vers Ben-mohamed Madi T	-660.00	e247cc582c7f8df788a3122cb497b496362dfea5e63f8e6bc887d8f529af036e	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
97	2	2026-02-17	Virement De M. Madi Toumani Ben-moh	250.00	08d15a6999b83aac4b0a65e381bebbfbc4f2d46842079730a807cfd5b57bd020	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
78	2	2026-02-03	Lessonsharing MTB-2026-01-30F:1-10-11-1	230.00	34e48e937b45fe10e38a4d3ef75541675a3f0f21b34f471481ae6e9281ad01be	28	1	LessonSharing	Lessonsharing - - -1	
204	4	2026-04-20	Carte X4519 Burger King Les Ulis 18/04	-12.95	6457babaf668fb80445de9d3e30f1e29ecdfcbe93aee14b4b72c57f5cb750963	24	1	Burger King	Burger King	
134	3	2026-03-09	Carte X4519 Deliveroo 75009 Pari 06/03	-25.30	0fe6caa612915c190ac3296281aa6345ff564b4b786668a4ebb98c9488d9fed1	24	1	Deliveroo	Deliveroo	
302	7	2026-06-10	Virement Vir Inst vers Ben-mohamed Madi T	-14.00	f6367cc71a5863274517d569ac92e53ab4264fcecc195c7c0b2f9eb09b388c1b	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
224	5	2026-05-11	X4519 Les Uli 07/05	-8.03	67b2eea7c089e8d51c81a4ef88470a90f3987c433b198a3e895e33207751f37a	24	1	Carrefour les Ulis	Les Uli	
206	4	2026-04-20	Prlv Idfm IDFM/001788508-3515435-1788199/MAND ATE-REFERENCE000000000000556877 Core	-6.12	d75626e5f0bb7736df2bc18ad43b4aeca696d77c06f92bb6981fa1ad023d1699	23	1	IDFM Core	Idfm Idfm - - Mand Ate- Core	
72	2	2026-02-03	Virement Lessonsharing MTB-2026-01-30F:1-17-11-1	1210.00	08e510e954adc3cb54a6ffb32d507353304685d92db5ad9691a833b5044fc25f	28	1	LessonSharing	Lessonsharing - - -1	
73	2	2026-02-03	Virement Lessonsharing MTB-2026-01-30F:1-01-11-1	920.00	da008674f141a826aea2db4f15fe7ec4573fb0a4089e0ffe8b1c25f260bc7db8	28	1	LessonSharing	Lessonsharing - - -1	
75	2	2026-02-03	Virement Lessonsharing MTB-2026-01-30F:1-27-10-1	1520.00	d765dd445a67674637d15baf2470e85755fe2796b5cef60c50d7b9e319f29904	28	1	LessonSharing	Lessonsharing - - -1	
308	7	2026-06-15	De M. Madi Toumani Ben-moh	605.00	c08dce2048516d42d084f14b55e65c6e65c4ecb29ed2d7b64f7dcf8479f6dc4c	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
2	1	2025-12-24	Virement Lessonsharing MTB-2025-12-22F:1-11-10-1	1520.00	f2713a5dacedcb167cf6f3eb49c8b549bfe3bd4c03aed4e91aac45751d5cbd7e	28	1	LessonSharing	Lessonsharing - - -1	
118	3	2026-02-26	Carte X4519 Mcdo Lohearches Vinc 26/02	-12.15	3fda4331500425f687b18050690c4329e7234719d7de6594cd4606f0d758ea0b	24	1	Mcdo	Mcdo Lohearches Vinc	
211	5	2026-04-27	Carte X4519 Sc.pedra Alta Paris 25/04	-45.00	b96c3a1eb5016bcd959f3534d928ccc64b35265d9803264d825f3ec699f29730	33	1	Pedra Alta Paris	Sc.pedra Alta Paris	
81	2	2026-02-03	Vir Inst vers M.madi Tarik	-100.00	5d007a65d275e443a5348b3e7d331e56dffa2468635fe2a11ee808fe39d35cbd	29	1	Vers Tarik	Vers M.madi Tarik	
130	3	2026-03-06	Prlv Acheel France REF-2F3E4B269AC60316D9GFGWBH4 Core	-12.47	e65ddc0780565b31e5518333df2e223287ee76a0f81318e6c95fff09d475adde	22	1	Assurance habitation Acheel France	Acheel France Core	
137	3	2026-03-10	Virement Vir Inst vers Ben-mohamed Madi T	-30.00	3518bf0190783499552487d438d91adfd60d40d7fa15058ff3de0c16bc415fb2	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
179	4	2026-04-03	De M. Madi Toumani Ben-moh	1850.00	ee6a0a2160a208cc640abea629b19b2a1c3c6fcede04b3fc806a2228a1a35424	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
282	7	2026-06-05	Virement Lessonsharing 2026-06-04-MTB 00002F:1-23-03-1	840.00	482a9a0bd04cb297d681136d203bfc4c76108a23d3e4706ed43fd71180cfa04d	28	1	LessonSharing	Lessonsharing - - - -1	
313	8	2026-06-24	Virement De M. Madi Toumani Ben-moh	80.00	7e734b83c289b3501bb7ec242455bfa4baa54338fd32a87d5962e8e03d4ab94e	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
315	8	2026-06-24	Vir Inst vers Ben-mohamed Madi T	-25.00	4a3335d994384aed4a0f0c9084917b33493df28b254c6fa3f8119d59b972322b	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
317	8	2026-06-25	X4519 Les Uli 24/06	-12.08	76060904d5d3ceab8f9a39a26e84c42784a37fcbaa633e1e54ef08e0230fa8bc	24	1	Carrefour les Ulis	Les Uli	
318	8	2026-06-25	Virement Vir Inst vers Aya N’goran	-50.00	3b6780cc34f88a4cbf4a175e1ba9854b565a9cc7194c1e58e8a7f1b70b429a25	33	1	Vers Sonia	Vers Aya N’goran	
319	8	2026-06-21	De M. Madi Toumani Ben-moh	160.00	5f41610932627f899d8896d671094a49f5c09ba197e6a72edca1ca5952e92b14	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
320	8	2026-06-29	Carte X4519 Action Les Ulis 27/06	-17.65	d20785dfe248c85a9af14d0ea110f099d1d22b32e7a84588978bd089d8c1074b	24	1	Action	Action	
273	7	2026-05-23	Virement Vir Inst vers Ben-mohamed Madi T	-22.00	3eec624ce043df646a7a82edd5cdf1d7eb1006ecb67dbef83c699543265ad80f	44	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
323	8	2026-06-29	X4519 Ubr”* Pending.uber.co 28/06	-61.90	cf9bcf0992b5431c1c0f78f51f19bebae0a5717c342f5820e961f24018857b3f	23	1	Uber	Uber	
185	4	2026-04-07	Carte X4519 Deliveroo 75009 Pari 06/04	-17.33	bc03b58722727e16313be42d36a7a44a4a39636d294238e7b0fef8af46f6814b	24	1	Deliveroo	Deliveroo	
208	5	2026-04-22	Prlv Paypal (europe) $S.a r.l. et Cie, 1049757239770 Paypal 567J2253W2NHA LU96777Z0000000000000000058 Core	-21.25	b6a2997e79492c20f4a480d1297d6d641c7b807634876eb03da64c381ff92218	26	1	Paypal	PayPal	
329	8	2026-07-04	De M. Madi Toumani Ben-moh	1300.00	aa20d2ba434f23761b6460c2675c752d457b03cc2f9de3f5b3906d016d4d0954	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
330	8	2026-07-04	Virement Vir Inst vers Mme Jing An	-950.00	e17df63ac9e7b9ae929b17b6a0fa659085e4e2d8fff3f119b71d2cb5b1368343	22	1	Loyer	Vers Mme Jing An	
331	8	2026-07-06	Prlv Free Telecom Free Hautdebit 148601832347939382 Core	-29.99	6037c8add6197b65de62eb4db7826dbbb58e3f7e8c46a1fddac68c1b0c5a0086	22	1	Free	Free	
332	8	2026-07-07	Vir Inst de Madi Toumani Sakina	800.00	7e493a568cda0f0c7eaf94c5e19859a22bce57f79ef39cebdaba96dcde6c82f5	29	1	De Sakina	De Madi Toumani Sakina	
333	8	2026-07-07	Prlv Acheel France REF-E65CAODEAABADO3246GFGWBH4 Core	-12.47	0e4de66dda8b8c1cd85017d58ca09196d1512a9b94af7389e168b67ea93abb2d	22	1	Assurance habitation Acheel France	Acheel France Core	
334	8	2026-07-08	Virement Vir Inst vers Madi Toumani Sakin	-500.00	e06f43e8d872b5cfc25856d1d96d9f93fdcb8fb77aaf21323c7fe375e6ce825e	29	1	Vers Sakina	Vers Madi Toumani Sakin	
335	8	2026-07-08	Virement Vir Inst vers Ben-mohamed Madi T	-150.00	bde62e7f5ec3ed48083d7f06663a22ac8e1eb1017fe2e7414ce8f3488eff4cfe	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
336	8	2026-07-09	Prlv Offre Essentiel	-6.00	852d7b3daeaf03cedf168610ba67b3ee2137f7b5f862853d73cea9b4ad15adc6	30	1	Frais Bancaire	Offre Essentiel	
338	8	2026-07-12	Virement Vir Inst vers Madi Toumani Sakin	-100.00	2386e1916017b7596e2531f86b2d164b55b883a6bb9b6f85908b213491ddbb60	29	1	Vers Sakina	Vers Madi Toumani Sakin	
115	3	2026-02-24	Carte X4519 Yamanashi Paris 23/02	-18.50	780217087928c12cce080b666bc6538eeabdbb00bb8f3f4791f0daa4c54ba92d	28	1	Yamanashi Paris	Yamanashi Paris	
341	8	2026-07-16	Virement De M. Madi Toumani Ben-moh	90.00	1fcddb5e0650757891ed68ea481f5e2f2632d3abd6e544a50b06214611f39b5c	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
342	8	2026-07-16	Virement Vir Inst vers Ben-mohamed Madi T	-33.00	d5f30007a05bd7eaabc80534183defcc4e30bd6b133d65ecad105f62cde60f52	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
343	8	2026-07-17	Prlv Engie Mandat Core	-54.88	c34e508ea07ee5546592bbae35c439e9c73d6d37cd9ac02d0142013614aebed5	22	1	Engie	Engie	
65	2	2026-01-28	Vir Inst vers Ben-mohamed Madi T	-25.00	5fd1ac8a39b9ca7719b69c0bad51a8351458cba8d701a5c8582b88afc2fe8e87	44	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
321	8	2026-06-29	Carte X4519 Buy Paris Duty Roiss 28/06	-19.99	c31ffa7f952b426bb74b42438675184afc4251fa2d19f26aace3c10e3dfac4bc	29	1	Buy Paris Duty Roiss	Buy Paris Duty Roiss	
31	1	2026-01-12	Carte X4519 Nyx*justeatemps Rois 12/01	-0.50	f55e2cfea00e105f48eb0394c223248c65247990d0fcaf22313f297df4cacc83	28	1	Machine à café	Nyx Justeatemps Rois	
38	1	2026-01-17	Virement De M. Madi Toumani Ben-moh	600.00	a5d0738b18a5e27b031508e85eb70119b02372a94ea0deefdd8d4a7c152243e3	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
104	2	2026-02-17	Prlv Engie Mandat Core	-54.88	3a080721448516d6ae2ede8f29a4375ba5e135fdf3d44f2f88550af230bc6c69	22	1	Engie	Engie	
170	4	2026-03-30	De M. Madi Toumani Ben-moh	100.00	0edec6a73df9c23c467f04bc00989d30dab691809423de94cdc96fc11789554a	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
200	4	2026-04-16	Prlv Paypal Europe $S.a-r.l. et Cie S. 1049622956887/PAYPAL567J2253W2NHA LU967ZZZ0000000000000000058 Core	-50.00	a2af1f539d6f13836e309a1ef7fb5b75b08e824bd512e3a669f9908c60ed31bb	30	1	Paypal	PayPal	
297	7	2026-06-09	Prlv Offre Essentiel	-6.00	1713b5f2cc2c34ddf3128af25d9d0a957b4a7a7232af99e5bfa5ef1244f8bb1f	30	1	Frais Bancaire	Offre Essentiel	
314	8	2026-06-24	X4519 Arcueil 23/06	-2.29	2bfac77b61f01e4ed01fdcc0a885af27972e82191e28e0ed8ccfffc98fb4e9e5	28	1	Gare Arcueil	Arcueil	
71	2	2026-02-03	Virement Lessonsharing MTB-2026-01-30F:1-22-12-1	1520.00	e5aa4b7a68b8bdce8c64c63f6381d977ed3e0da02d1939b488a0af0ee8ce8a88	28	1	LessonSharing	Lessonsharing - - -1	
87	2	2026-02-08	Vir Inst vers Ben-mohamed Madi T	-100.00	0fae0d56dac69a70c4741d79c9c8d7f4e229dc51df3367219b57f774ea038106	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
164	4	2026-03-24	Prlv Paypal (europe) $.a r.l. et Cie, 1049085182258 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-162.50	a6477adcc06769ff91f216f650ea9c2d1e9c6e6a6f872aac874c50a9c4b09b52	30	1	Paypal	PayPal	
322	8	2026-06-29	X4519 Les Uli 27/06	-19.34	221a2feafd0f7c8190d663e757a5ba023ca2d3e44c8ae2bc1d71e7fecd33fd3c	24	1	Carrefour les Ulis	Les Uli	
396	10	2025-12-08	Carte X4519 H.t. Food Les Ulis 06/12	-14.00	c2c53e6b57e5ab07283bca3e0c556dc5e16db12749a7fc38d81cc96228241949	24	1	Les Ulis Food	Food Les Ulis	
280	7	2026-06-05	Virement Lessonsharing 2026-06-04-MTB 00003F:1-26-03-1	840.00	33b68fda05ed7ed46ef671d4656513eeb987332a6a1a3068a6a951c7b63134b3	28	1	LessonSharing	Lessonsharing - - - -1	
397	10	2025-12-08	X4519 L Empire Du Costume 06/12	-270.00	2458545596b8b72e711cf18a7500c14d6d1fbbb1bd0ceb53a7bd8012915a89f4	30	1	L'Empire Du Costume	L Empire Du Costume	
379	10	2025-11-23	Vir Inst vers Sakina Madi Touman	-100.00	eccbf8f87dcbda2d5e8475ec87bc6b66d700a411228e29a29f48c1d3035bd891	29	1	Vers Sakina	Vers Sakina Madi Touman	
383	10	2025-11-24	Virement Web Clement Depernet	-30.00	9edc728628f1933f68891aa95b57f5f92dccca64284f87a6cc2e1c00e24a4480	33	1	Clement Depernet	Web Clement Depernet	
377	10	2025-11-22	De M. Madi Toumani Ben-moh	50.00	a8746ff131ffb7c32c9a3253e9a202646074393312a36e53431d4f9a45e6bf02	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
378	10	2025-11-23	Virement De M. Madi Toumani Ben-moh	100.00	b7b32b504efb7c65e63a1228f435e5f06426c4733ffe3e3d41b1fd1a9b0af65c	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
380	10	2025-11-24	Virement De M. Madi Toumani Ben-moh	30.00	ebaedca1d0643a41c5b637c437e6ccfee487d45bf6265e5403d0a36b467a2694	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
326	8	2026-06-30	Vir Inst vers Ben-mohamed Madi T	-100.00	07f143651b2a1bd1324823de7bbc13f8de13a8ea6d48a1890b9192303286b774	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
327	8	2026-07-01	Virement Mens.pel 44690429796 07/26	-50.00	9808fbd391978d6f8d8a52da470640b6ddedecea486d618a01172eee1e864bc4	27	1	Versement PEL	PEL	savings
328	8	2026-07-02	De M. Madi Toumani Ben-moh	150.00	57a9ceeae2177469d2c7ffcd5fd71771273b58b866847e5d819ddb3dc20b2dc7	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
386	10	2025-11-29	Vir Inst vers Ben-mohamed Madi T	-70.00	69f20336da6f8a224cd3e73d079b39bb439926cc96373d62866b064831bdf16d	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
387	10	2025-11-30	Vir Inst vers Madi Toumani Sakin	-30.00	d909c113ef4f9434691803559d0dc833259b735539a8381e00670ad44df4bd75	29	1	Vers Sakina	Vers Madi Toumani Sakin	
388	10	2025-12-01	X4519 Service Navigo Paris 28/11	-22.20	82ef6a83c74c47fbcb9bb5d9aab4175046851f50b702b07f5f822e85b765938e	23	1	Service Navigo Paris	Service Navigo Paris	
389	10	2025-12-01	Virement Mens.pel 44690429796 12/25	-50.00	12f9799e9cafac58ae2763299ef6f29e9f7cdb9d08f15ab09e41b18281df7fc7	27	1	Versement PEL	PEL	savings
340	8	2026-07-15	Virement Vir Inst vers Ben-mohamed Madi T	-50.00	c9e8183b881a5b96b4de9b7f069621317f7674d041f64a3d83db46a7abf19b7f	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
392	10	2025-12-04	Vir Inst vers Ben-mohamed Madi T	-140.00	8795dcd12f834f0baecf9d3804e01f3cac5240e285beb0b3c922be2a237b33a5	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
393	10	2025-12-05	Prlv Free Telecom Free Hautdebit 143414470847939382 Core	-19.99	b5a06b98ce6c12894142f18e9b182b5df3a2011d3db87683d931bdb62dd59818	22	1	Free	Free	
394	10	2025-12-06	De M. Madi Toumani Ben-moh	350.00	37e95e7f720990581e128dad0d21a7378109964162847400959ce4f8b241fcff	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
395	10	2025-12-08	Carte X4519 Deliveroo 75009 Pari 06/12	-30.29	d1f195bf94296090fb9bfa5d98c70b3e40b456ff8f1ae79a2c2e7f42b3dcc47d	24	1	Deliveroo	Deliveroo	
400	10	2025-12-09	Prlv Offre Essentiel	-6.00	52308d18cee3fedc4a26ca5fa0c0440bd9b07ebc0e3f5b809c399c0df612abe6	30	1	Frais Bancaire	Offre Essentiel	
401	10	2025-12-09	Carte X4519 Sncf-voyageurs Paris 08/12	-30.00	88ebc9b859559a5af4ea4c9e700b3cd64f8b5d86998bd1301194084557dd4ed9	23	1	SNCF	SNCF	
403	10	2025-12-10	Virement De M. Madi Toumani Ben-moh	200.00	94f77c3e8ee0798430990fa07807982028193b22d2fc001f82fc7d16c59d5edc	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
404	10	2025-12-10	Carte X4519 Burger King Les Ulis 09/12	-14.40	9180f34a30765798c326517b74b8545af8063f435399d0ce6ad4aa78fe6731d1	24	1	Burger King	Burger King	
407	10	2025-12-10	Vir Inst vers Madi Toumani Sakin	-50.00	344d0e56ee8c7cb43486808575dc5b5386947a650e77ea3413dbe0fcfab3aaf4	29	1	Vers Sakina	Vers Madi Toumani Sakin	
408	10	2025-12-12	X4519 Africa Exotique Les 11/12	-10.32	808c3fcc8cd9a2a0a8ff61e0d8fed45141e8817e78fee0e1fd75276c33c878b7	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
409	10	2025-12-12	Carte X4519 Ulis Viandes Les Uli 11/12	-12.45	cb8c44fb415c6a048144e85c130cbcc890fe671c7aaad2bffbbbf7d804e63fad	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
410	10	2025-12-14	Virement De M. Madi Toumani Ben-moh	100.00	22fa83ad50aca358aadbc433e705fac12d4a220b1428777d52ec9ef6cf4ebf30	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
411	10	2025-12-14	Virement Vir Inst vers Ben-mohamed Madi T	-100.00	530a2050b95e764d4cf6b2ce58549d81061fa83370e5562d5366575ee77d6b82	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
412	10	2025-12-15	X4519 Lw*sncf Connect Pari 13/12	-17.35	fc3f6d80a562abcaa571896d517c818f5b9b59c897ac59542088fdc6a59c275d	23	1	SNCF	SNCF	
324	8	2026-06-29	Ret DAB X4519 Les Ulis 2 Ga 27/06 09H31	-20.00	8e40defb0314ec216ff27b26600f4755fe46e260b50fcb2e60c38a3b9c95fbd9	24	1	Retrait Distributeur  Les Ulis	Ret Les Ulis 2 Ga	
429	11	2025-10-22	X4519 Ratp Paris 21/10	-4.50	0f9b97af9a72aa70f6da9824f6072dee9d7dca75b0ba9e7737298ddfeec9c17e	23	1	RATP	RATP	
435	11	2025-10-24	Vir Inst vers Ben-mohamed Madi T	-10.00	7655d9bfac9b11da5c6156c9c67c8c963b2de0b670f4c24d866e7c272ca454e2	44	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
437	11	2025-10-27	Carte X4519 Laposte L916920 Les 25/10	-9.91	f4c270a7e210e8f5dd98712fdd24985ca1d310fe1da811ef8730bfcd3b7ecaec	30	1	La Poste	La Poste	
413	10	2025-12-15	X4519 Uber “trip Amsterdam 12/12	-12.95	226857ee0e9d7fd80a40ec453f395613f5d5cac129e8514e055ad2e3351c985d	23	1	Uber	Uber	
416	10	2025-12-16	Virement De M. Madi Toumani Ben-moh	150.00	c0153b363143eb011c476e284c5fceb0bedccca1bb1e6c28195790f4c754c9d6	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
418	10	2025-12-16	Vir Inst vers Aya N’goran	-100.00	9d17cd3131dcbc08970421b580e8ea272b6e1c3ea58c7f063fbdea92ced9ecc4	33	1	Vers Sonia	Vers Aya N’goran	
419	10	2025-12-16	Virement Vir Inst vers Madi Toumani Sakin	-150.00	a5fdf564b371ecf8bbb36ebe76f739c4ed7b427f9638391d9a9cb74ac4cd8a86	29	1	Vers Sakina	Vers Madi Toumani Sakin	
420	10	2025-12-17	Prlv Engie Mandat Core	-47.03	b06a93bb400abdaa944a15d18d74a8539aabdc5eb9492c423cab5c96788b9f99	22	1	Engie	Engie	
421	10	2025-12-20	De M. Madi Toumani Ben-moh	150.00	e6fcd7c0a0827329ea95e1f7311d19e20135bfb3d4f8a988991bce3bd526daf6	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
423	10	2025-12-21	Vir Inst vers Ben-mohamed Madi T	-13.00	1d7cbaaad337ca94025182472c77dba5216fa8c4e12763132a9e11ae18d5c0b8	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
424	10	2025-12-22	Virement De M. Madi Toumani Ben-moh	190.00	247b255c3f0e496ffb431da2bd3f83869d17b38c42805bd68fd6d6c641c68ccc	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
425	10	2025-12-22	X4519 Les Uli 20/12	-83.70	63a1ec8334ab0d05bf0178cf62709368836ff59124858498009900a5265d340e	24	1	Carrefour les Ulis	Les Uli	
426	10	2025-12-22	Vir Inst vers Ben-mohamed Madi T	-5.00	d9d2c390c31e27a4b7af9375c2f85b3df4f863a59391200fdf941c33c3de2c76	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
427	10	2025-12-22	Vir Inst vers Ben-mohamed Madi T	-190.00	1ef49d563b53db599b42f5224e499cd64998ff356d77ab3bd684a5baa98b1a65	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
438	11	2025-10-27	Carte X4519 Nya*juste A Temps Pa 24/10	-0.50	cffafc663b4ef5bd93c371bdaff94960883df96910183ad850e1b9beb9f70481	28	1	Machine à café	Nya Juste A Temps Pa	
382	10	2025-11-24	Carte X4519 Les Uli 22/11	-71.53	e290eae03b1f4979777f4d87982f496990e89f8dd1d5cd508bf7d0085e0ae8c7	24	1	Carrefour les Ulis	Les Uli	
384	10	2025-11-28	De M. Madi Toumani Ben-moh	30.00	5582f57d84834dd753191ba2740f9f2bc561985fc79ee4a380949affe0a84807	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
385	10	2025-11-28	Lessonsharing MTB-2025-11-28F:1-20-10-1	560.00	d70bb435cd9a63eae7d5fc20f89fd883565570b2aaf0b00a0e0cb51c6fdfe68a	28	1	LessonSharing	Lessonsharing - - -1	
391	10	2025-12-02	Vir Inst vers Ben-mohamed Madi T	-750.00	151404e7216b597d4c9e5034f3e30888cb7a6bbc72b996970b8cc2aa69fd84fa	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
442	11	2025-11-01	Virement De M. Madi Toumani Ben-moh	20.00	95fc2878baba143c87c5b489d9537392c9f43a73e07e107818e07f82a3f73f42	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
443	11	2025-11-01	Virement Mens.pel 44690429796 11/25	-50.00	1589ce187379c20cbf53b478fb16f1cfcf922eed79af37edb138168d9454ec62	27	1	Versement PEL	PEL	savings
444	11	2025-11-01	Vir Inst vers Ben-mohamed Madi T	-8.00	02065de4cfbac70ae71e8c3682d67ad72eee3e0435d2c760492af82a8cd22c49	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
445	11	2025-11-03	De M. Madi Toumani Ben-moh	1200.00	9ca028ef335e17f5523f2014102ec5dfff602fc2ab31dfc9149a5f63960ba78d	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
447	11	2025-11-03	Lessonsharing MTB-2025-10-31F:1-09-08-1	1520.00	8eb3376de3845a505613f003908e1a9d8c4c457029c691ca1031ba275d909839	28	1	LessonSharing	Lessonsharing - - -1	
448	11	2025-11-03	Virement Lessonsharing MTB-2025-10-31F:1-14-05-1	1210.00	434819073c5208ebfa06ad6a1fe6bb2b700f1eba673afd7781655672352f0d71	28	1	LessonSharing	Lessonsharing - - -1	
449	11	2025-11-03	Virement Lessonsharing MTB-2025-10-31F:1-13-09-1	840.00	ac3f7afd94cb242a3e947c95602c5f30dda8a8c637adeac4797802a7649a3a01	28	1	LessonSharing	Lessonsharing - - -1	
450	11	2025-11-03	Virement Lessonsharing MTB-2025-10-31F:1-10-09-1	460.00	583bf21fc7bd2ed09f918da4bd153703be1d1d590fc16b2be4ff28aefe4154bf	28	1	LessonSharing	Lessonsharing - - -1	
451	11	2025-11-03	Virement Lessonsharing MTB-2025-10-31F:1-01-09-1	1520.00	4c5f9840cc80337e79182374e2a3ff603a9a31c7678f2771868acc56de273649	28	1	LessonSharing	Lessonsharing - - -1	
452	11	2025-11-03	Virement Lessonsharing MTB-2025-10-31F:1-26-09-1	1440.00	0e72a429d52c3b17cdc697ec274e0bf51b3c4a0f72cdbb7398aecd6ba8853f30	28	1	LessonSharing	Lessonsharing - - -1	
453	11	2025-11-03	Lessonsharing MTB-2025-10-31F:1-02-08-1	1510.00	4ff171ba6e8d5700d2a20bcc393c0b148034716ad7d9c02a6e962bc1670b5283	28	1	LessonSharing	Lessonsharing - - -1	
454	11	2025-11-03	X4519 Les Uli 31/10	-39.11	f6e2de1dd5cfaf4e06f82cc92c9f26e9695d8cda57470c614439f54b03a11a67	24	1	Carrefour les Ulis	Les Uli	
492	12	2025-09-26	X7303 Paris 25/09	-0.70	3ef537783856cc144bac6b6711b0c3b3c8b6b8fa299a3ed1173b8a682687b07e	28	1	Machine à café	Paris	
455	11	2025-11-03	Carte X4519 Service Navigo Paris 31/10	-22.20	4e8b0d7e5f3ba50dea6d26f2c1a5aa529c180be076eac58530eec847d0cc9bfc	23	1	Service Navigo Paris	Service Navigo Paris	
467	11	2025-11-10	Carte X4519 Ornikar 75019 Paris 09/11	-7.99	e31eddce3b6e3910a4ea1bcead291190f4de6e1733288bb8775661e78032ea4e	29	1	Ornikar Paris	Ornikar Paris	
471	11	2025-11-14	Carte X7303 Amazon Eu SARL 14/11	2129.00	f633accd59addf860075ba132e9a229ab846e85f7b9745fd70cc397f6c36c756	30	1	Amazon	Amazon	
441	11	2025-10-27	Carte X4519 Ubr”* Pending.uber.co 24/10	-32.91	5e780929e23c09ec3b7bfde74aadff9306dbc00ff66824d87638ab1641c31052	23	1	Uber	Uber	
414	10	2025-12-15	Ret DAB X4519 Cesson Sevigne 12/12 21H53	-80.00	6f70e0355de144bf265ea97c3190f8d7127a97c172e31385a3d79a6a93dc1615	30	1	Retrait Distributeur Cesson	Ret Cesson Sevigne	
472	11	2025-11-14	X4519 Binance.com Vilnius 13/11	-15.00	683762045ba99955b7badc2ab1d306eb2f9b57623dc3acf4754a1b8e4062e588	44	1	Binance	Binance.com Vilnius	
456	11	2025-11-03	Prlv Paypal (europe) $.a r.l. et Cie, 1045862969611 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-244.80	dc8f75fa8844d88f2bfb5d6ade575c7d027254d3f60e44114dd5a4a9afc6836a	26	1	Paypal	PayPal	
457	11	2025-11-03	Vir Inst vers Aya N’goran	-1200.00	9b8e6579e570f64b1a81a6d031f19978b83a60fbd0744bce6580362d91f2e4a5	33	1	Vers Sonia	Vers Aya N’goran	
458	11	2025-11-03	Vir Inst vers Ben-mohamed Madi T	-500.00	d02331b696286f7a99f5899768e342731d209aa5bbcef6a0c39409734943f55a	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
459	11	2025-11-03	Web M. Madi Toumani Ben-moha	-50.00	fbd409401b6bfe6590ba8d522d8e534c51510a29900c67b90b285df38d10b984	31	1	Vers Livret A	Web M. Madi Toumani Ben-moha	
460	11	2025-11-03	Web M. Madi Toumani Ben-moha	-6000.00	dcc6df3a53902af883f3d7b5305356835f3139019f6ade4949f8a4cdeb66b8c8	31	1	Vers Livret A	Web M. Madi Toumani Ben-moha	
461	11	2025-11-04	Carte X4519 URSSAF Marseille 03/11	-2485.00	6b685096b3bfc72adb5e09e8095b9f8a28fabd6b87eb1949907ca5d90ef46d84	28	1	Cotisation URSSAF	URSSAF	tax
462	11	2025-11-04	Carte X4519 URSSAF Marseille 03/11	-635.00	618fdf6bebfc870350d529946ec4104b35c1390fc66b8f57aa93bb31fd6ae2a1	28	1	Cotisation URSSAF	URSSAF	tax
463	11	2025-11-06	Prlv Free Telecom Free Hautdebit 142655779247939382 FR83277459654 Core	-22.98	0714cf31ecf04d8f7ee1e5a682a52a54d53d8dbea6786a01a70c4365bdacf7d9	22	1	Free	Free	
464	11	2025-11-06	Virement Vir Inst vers Ben-mohamed Madi T	-50.00	68982bbc00702f4cce9a7b8c6385664da70e9e3188c77043cf1db0784ede9629	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
466	11	2025-11-09	Prlv Offre Essentiel	-6.00	ea1e58fdb5a4d8f490d814490eff50fb1e544fa30ccf20f2171cc29276279dc4	30	1	Frais Bancaire	Offre Essentiel	
498	12	2025-09-29	Carte X7303 H.t. Food Les Ulis 26/09	-10.40	daf6d61dff0efd18117eaf1c0f63fdd42e2f24ef5755c717f29445e6896f1873	24	1	Les Ulis Food	Food Les Ulis	
484	12	2025-09-23	Carte X7303 Paris 22/09	-0.70	c778dd529feeda6202e7d6c1d6a98346358f8f2ce0dfd06287f645f1537f1ed2	28	1	Machine à café	Paris	
469	11	2025-11-13	Carte X4519 Deliveroo 75009 Pari 12/11	-29.32	ae4f41f18ab7660aeedc2a665c251b41473fba983794039b5bda4928d9186f6e	24	1	Deliveroo	Deliveroo	
470	11	2025-11-13	1 13.11 Vir Inst vers Abdou Madi Toumani	-100.00	a9819a629a9ef6977128ee1e8c6137c7ef92539fb5f350e24572144ca49871d7	29	1	Vers Abdou	1 . Vers Abdou Madi Toumani	
433	11	2025-10-24	De M. Madi Toumani Ben-moh	300.00	742ce226559c3d36bd3362625eaa610206c37bacd546e7219b2092dac119dc6c	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
434	11	2025-10-24	Carte X4519 Nya*juste Temps Pa 23/10	-0.50	f0b2c1ca7a437fdf4d4d87189055295b63ad2600532574380c4f309f64922188	28	1	Machine à café	Nya Juste Temps Pa	
436	11	2025-10-27	Carte X4519 Action Les Ulis 25/10	-15.35	1fe339b86a9e7a7cd2f71f3bb2fb8c6b2c2547efacd6359a3c55a48d6d8b95ca	24	1	Action	Action	
485	12	2025-09-23	Carte Ulis Viandes Les Uli 22/09	-13.63	e084a2d15b7ba48ed9701102a9ee0a33c1e982b17eefc20c478f606c473d0afb	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
486	12	2025-09-24	Carte X7303 H.t. Food Les Ulis 23/09	-10.50	477585203a32c6fcbc47da5009e26345359a1c0a746b331492dc2cc9fe4a0f86	24	1	Les Ulis Food	Food Les Ulis	
491	12	2025-09-26	De M. Madi Toumani Ben-moh	105.00	aca890dceabebe63a6ff446fce6e8f8b63aa2aa24bae78e2d1decfecc9bf7aa6	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
497	12	2025-09-28	De M. Madi Toumani Ben-moh	500.00	9c4a9fb0a4d7eeb94f362cb7b45f4239308d9013aab79fb96e9cfb182eb10ba3	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
496	12	2025-09-26	Virement Vir Inst vers Trade Republic Pea	-105.00	205c01a9315b15110949444a8ab3380f3738aa6d74c30ee01b01f1dbd912415e	29	1	Vers Trade Republic	Vers Trade Republic Pea	
480	11	2025-11-21	121.11 Vir Inst vers Ben-mohamed Madi T	-25.00	096754cc0f11fbd86b8cd85c74d9f0de2e6f9d20bcd3f52173bdb3afe641f69c	44	1	Vers Trade Republic	. Vers Ben-mohamed Madi T	
499	12	2025-09-29	Carte X7303 Service Navigo Paris 26/09	-22.20	85d16a82fcc7629bc3a1bc8488e917913954b48da9394a6dc4de54dfade2b91f	23	1	Service Navigo Paris	Service Navigo Paris	
500	12	2025-09-29	Carte URSSAF Marseille 28/09	-454.00	bc5cab2281e7fc282775bedf669105b48445615007bf784f40cb925c73d1ef1e	28	1	Cotisation URSSAF	URSSAF	tax
468	11	2025-11-13	1 13.11 De M. Madi Toumani Ben-moh	650.00	19bf97d18c489be3471d41527231a5083c7ced39e89613bb95e87a48b17c5605	31	1	Vers Compte courant	1 . De M. Madi Toumani Ben-moh	
502	12	2025-09-30	Carte Nya*juste A Temps Pa 29/09	-0.50	22f9b65e83125390cd1eaa7e199b7bc9bbeec4b09c64efce1a78eb47c387d25f	28	1	Machine à café	Nya Juste A Temps Pa	
503	12	2025-10-01	De M. Madi Toumani Ben-moh	900.00	c6add3449ae470a280f161ee7e0263c3b260daebede317183b0ec3483c36d106	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
504	12	2025-10-01	Lessonsharing MTB-2025-09-30F:1-03-07-1	560.00	27d2bd383d95ae2b4318d26d6462dbac7d6d57f2a18edb215bfbf714d793dc52	28	1	LessonSharing	Lessonsharing - - -1	
505	12	2025-10-01	Virement Lessonsharing MTB-2025-09-30F:1-20-06-1	560.00	9be67d2736bbd627f9a73ba01b1b3d99a66c62f3f8f2b063a43a2384a3706002	28	1	LessonSharing	Lessonsharing - - -1	
506	12	2025-10-01	Lessonsharing MTB-2025-09-30F:1-05-07-2	290.00	7665879be18ba6854cf7df74046e0a35c6e629e09578d1af4471ec67cb799f26	28	1	Lessonsharing	Lessonsharing - - -2	
532	13	2025-08-22	X/303 Maison Nouilles Pari 21/08	-15.90	ae6d02bdb219eea42068e641ff28bc4696f52fc183a9ced19c163ab667c8b474	24	1	Nouilles Paris	X Maison Nouilles Pari	
488	12	2025-09-25	Carte X7303 Paris 24/09	-0.70	3bef2c7cdbd5dc3ab09b57db303990d81831462637f8c904b7d9e0c790b5db29	28	1	Machine à café	Paris	
508	12	2025-10-01	Virement Lessonsharing MTB-2025-09-30F:1-05-07-1	310.00	e48098e059f43201cd984f45fffaa5205896bbc40bdfecd1382ff577a124eff2	28	1	LessonSharing	Lessonsharing - - -1	
509	12	2025-10-01	Virement Mens.pel 10/25	-50.00	7be9db45b1b242132d51cc75fc8881935d9541d44bec029b5c792cacf9fe609a	27	1	Versement PEL	PEL	savings
510	12	2025-10-01	Virement Vir Inst vers Mme Jing An	-2850.00	a5703d5d99f0e68172b07a1f84ca483dfcd71285f57f94fd0740204e666e7a36	22	1	Loyer	Vers Mme Jing An	
512	12	2025-10-02	De M. Madi Toumani Ben-moh	200.00	a2636b3a3acfef82c1fcdb16efe6cbc935b7262b317ec80c6f3b13bd196caf94	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
501	12	2025-09-30	X7303 Les Uli 29/09	-8.18	3748ffe10a9104a35715ad6e349393c04104633ae015d9b2e77010b6ff6519c1	24	1	Carrefour les Ulis	Les Uli	
515	12	2025-10-06	X8028 Mp”carrefour Les Uli 04/10	-47.53	417f45a5defec276471751fb1b8fbe147e48640f6775bfbc8f6f0b42dd06827f	24	1	Carrefour	Carrefour	
516	12	2025-10-06	Prlv Free Telecom Free Hautdebit 141894895847939382 FR83277459654 Core	-19.99	aa7bc8aa92332842a2cdd4394bfc33e0f42d3985cd096b707a0676db544f4ebc	22	1	Free	Free	
518	12	2025-10-09	Prlv Offre Essentiel	-6.00	492a70ad44eafe26c9ea5cfee172a84079ca2496af929ae5b9736457caa7e987	30	1	Frais Bancaire	Offre Essentiel	
519	12	2025-10-11	Virement De M. Madi Toumani Ben-moh	400.00	345a7fcf4fa0750520a7ad2a9b4883720b0865f918b4556e68a080e60bbc6a2f	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
520	12	2025-10-13	Virement De M. Madi Toumani Ben-moh	80.00	5243c40b2cb366610585e78cc07dfdec506ef0e4daa8d0ebe883d3c85ca06971	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
521	12	2025-10-13	X4519 Les Uli 11/10	-78.27	22414b9c3a5cea69d1b6aa183ffb7be1df8f69ea79ad1899dcc43ca98172dfed	24	1	Carrefour les Ulis	Les Uli	
522	12	2025-10-13	Carte X8028 Sncf-voyageurs Paris 11/10	-237.00	e2073e57bfa541e6432a0a659037d7e293daf4ef415d81cbb991751237f8eb80	23	1	SNCF	SNCF	
524	12	2025-10-14	Virement De M. Madi Toumani Ben-moh	100.00	310844e0cd196f58ed49f8bf8fd210c937cafec08049f6388e40732fbe6dff1d	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
525	12	2025-10-14	Carte X4519 Deliveroo 75008 Pari 13/10	-28.32	c275931e405228fc0457a1ae3a2b355944ec0ee6256bf63befd839dc703f2ec5	24	1	Deliveroo	Deliveroo	
477	11	2025-11-17	117.11 Prlv Engie Mandat Core	-47.03	287f6143a24c0cc292e89df945b30ed897ef253c9ff9446166178e2323768c66	22	1	Engie	Engie	
529	12	2025-10-21	De M. Madi Toumani Ben-moh	100.00	36865e132f0869c71501f869f472685302b3368c8ea93d9f4333dc1c52035e6c	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
530	12	2025-10-21	Carte X4519 Nya*juste A Temps Pa 20/10	-0.50	1f2b85ec84ecc7fbb64526616221da57ccd8d35bc0f4f4ae01ca3ee37116138a	28	1	Machine à café	Nya Juste A Temps Pa	
533	13	2025-08-23	De M. Madi Toumani Ben-moh	50.00	658215042bff7587fa0448a4df38853a68a3101d6f949979db0ffbc2d1ab1a7a	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
535	13	2025-08-24	De M. Madi Toumani Ben-moh	50.00	cc00ca0a9f5f3b89cecead489396492d0daa81733ddd099903f8251f3238b0b5	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
478	11	2025-11-21	Virement De M. Madi Toumani Ben-moh	180.00	7a028e1d81123d266bdd413aab9996f917ee9ba8d7791d91b9e8ac60f90177d5	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
482	12	2025-09-23	Carte X7303 Africa Exotique Les 22/09	-12.62	a13667885d71052ad95669ad8e5b5ecec0631a16c38e79c3889e5efc43acbbc3	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
988	21	2026-08-21	Vir Inst vers Aya N’goran	-750.00	7e8feb526c52d48ef1fa66b1df8d3d0bd53a107dc3619dc3466be66551126187	33	1	Vers Sonia	Vers Aya N’goran	
576	13	2025-09-17	Sodexo Coeur De Pari 16/09	-16.49	85a46469c2e86144373c55062a11fd36be7b9d68990a4b811a3f853b126d8a74	24	1	Sodexo Coeur De Paris	Sodexo Coeur De Pari	
539	13	2025-08-25	Vir Inst vers Trade Republic Pea	-115.00	33d4eea8c86c10538998a15aa7bc8b65971ac72ab1b3acdd1acbe37bd4a8466b	29	1	Vers Trade Republic	Vers Trade Republic Pea	
528	12	2025-10-17	Prlv Engie Mandat Core	-47.03	7fba0151011398c811d0ecd97963cf89ce7817f6273176b0de926a6b11347d8f	22	1	Engie	Engie	
569	13	2025-09-15	Carte X7303 Action Les Ulis 12/09	-7.24	40f6645807d9ddbecedaadb0793ff36d90686836b80ca3078f3a9cdb7ce3de80	30	1	Action	Action	
544	13	2025-08-29	De M. Madi Toumani Ben-moh	150.00	9ce0737927df27200de5dd59139665d365cb2a8c94bbe27fb8084db0c655fe60	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
545	13	2025-08-29	De M. Madi Toumani Ben-moh	22.00	c0b66496d5ddde7eafa41fc1baf6f9933087d2c45f8952dc48cf4df74d5887c2	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
548	13	2025-08-30	De M. Madi Toumani Ben-moh	50.00	4d0df523d708cacd4f8e4c65548f6beb83239664ed9cb5bc5d6963a3e33aa3a2	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
549	13	2025-08-01	Virement Lessonsharing MTB-2025-08-29F:1-18-06-1	1210.00	ca43389600204be665c8dc900464ad1e48149fddde005fc375b3a9e670e63dc3	28	1	LessonSharing	Lessonsharing - - -1	
550	13	2025-08-01	Virement Lessonsharing MTB-2025-08-29F:1-26-06-1	620.00	8a0d901659ba8f4c9692c66286ad56ea0c3decdb51a099690f88a4ec86651c30	28	1	LessonSharing	Lessonsharing - - -1	
551	13	2025-09-01	X7303 Africa Exotique Les 30/08	-11.53	27f8d0a44fac8e8418ca134aa88e0b7ee2067d14082680170900043032c18883	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
554	13	2025-09-01	X7303 Service Navigo Paris 29/08	-22.20	95f7590a95ccf737f31713717df0425e0c005257c2067d9ecfa3da01413b2707	23	1	Service Navigo Paris	Service Navigo Paris	
572	13	2025-09-15	Carte X7303 Sushi One Les Ulis 12/09	-11.41	9e1a518fa8f41a0f45e14e18518cbbd0fdc117916bc377ecc82fa5d6c4cc2ae9	24	1	Sushi One Les Ulis	Sushi One Les Ulis	
556	13	2025-09-01	Mens.pel 44690429796 09/25	-50.00	e03c6cc3edb23c6f6202841338ad7023f2ec81def78eaa58e18d6a9e330fef4c	27	1	Versement PEL	PEL	savings
558	13	2025-09-01	Virement Web M. Madi Toumani Ben-moha	-1000.00	fa5746e8c9c26226cadcf7fb1eb73990da88ac22f23b9851e01fb341783803cf	31	1	Vers Livret A	Web M. Madi Toumani Ben-moha	
560	13	2025-09-03	De M. Madi Toumani Ben-moh	400.00	6d0657ec13c399641dacf2671de5f02aef0d945099f72b8b9ee63c0b3356c92b	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
562	13	2025-09-05	Prlv Free Telecom Free Hautdebit 141140862447939382 Core	-19.99	01711646260209dd87fabdf225a4c6a938a94a73d97367bbc3a9f761c892a272	22	1	Free	Free	
564	13	2025-09-09	Prlv Offre Essentiel	-6.00	965b4dd49e7e2d0682a7647abaf68ecfd67340c51d9c418078f5fa21d8b320d7	30	1	Frais Bancaire	Offre Essentiel	
565	13	2025-09-09	Carte X7303 Burger King Les Ulis 08/09	-14.10	22d6a7e6474acbb930c0eac422413a6a1049d8708f3d08f890004ef403446a12	24	1	Burger King	Burger King	
566	13	2025-09-09	Prlv Acheel France REF-E8D84FAE441003DFFDGFGWBH4 Core	-11.44	e4a9e5c77d62748c901040b324ef87db2abc988b75e7e9796b5d9c459464e388	22	1	Assurance habitation Acheel France	Acheel France Core	
567	13	2025-09-12	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-13.99	8b3a451c6c350f0baeaee8c998695f87f72e7ab10a08a043b509b93355f8d19b	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
571	13	2025-09-15	X7303 Les Uli 12/09	-47.00	e68b74b2ec0c29f69887dbb889fedf9ef77b6585277cd60766281377e5790ab7	24	1	Carrefour les Ulis	Les Uli	
570	13	2025-09-15	X7303 Africa Exotique Les 12/09	-12.13	f15eaa9030f064a4d965e49d4cb56091f56ebd6d78ffdc0a3f71cdc3938ec70d	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
574	13	2025-09-16	Carte X7303 Mihir Et Associ Les 15/09	-31.73	430c18a02b2f80b46fb04fb5e8b9c892f83a1a123be98123100601a6b4d3720e	24	1	Mihir Et Assoc	Mihir Et Associ Les	
542	13	2025-08-26	Carte X7303 Ulis Viandes Les Uli 25/08	-13.60	7770477f126565839a1b905cc95ef7fd6bdd329440e4bf69e649167fd1c7cce6	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
555	13	2025-09-01	Carte X7303 Ulis Viandes Les Uli 30/08	-14.94	3e200d2e48efa17909e5d2b34f51b7d3dea94d0604cca486c78383c6200149c5	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
577	13	2025-09-17	X7303 Sodexo Epicerie Pari 16/09	-1.15	dab58f5dbb14e6e63e0bfae5887de9915dd17529a494da2a916f7745479c1ddc	24	1	Sodexo Epicerie Paris	Sodexo Epicerie Pari	
552	13	2025-09-01	X7303 Les Uli 30/08	-56.75	6cf446eff5a3196ea777cce2f38ceed8e2831ece02a27b488e399d55e069038c	24	1	Carrefour les Ulis	Les Uli	
575	13	2025-09-16	Carte X7303 Paris 15/09	-0.70	f6926561328e8bd08a5501434eed85da98be0469615aabe86bab34b2ad2558c9	28	1	Machine à café	Paris	
578	13	2025-09-17	Prlv Engie Mandat Core	-25.65	4ec0649698fdb706b8fb4493f5a74c1d96261d87b5707bac34489e88ec53a437	22	1	Engie	Engie	
606	14	2025-08-04	Carte X7303 Titanic Massy 01/08	-8.50	6d6bacc589c27718feee4eea7696bf41e86c9568e37022056acb604be017bf41	24	1	Titanic Massy	Titanic Massy	
580	14	2025-07-21	Virement De M. Madi Toumani Ben-moh	40.00	83df4cdc45494cbace090a68912f63b70dbe074176aa9bf0f2d061adbdfbee7f	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
582	14	2025-07-28	Carte X7303 Service Navigo Paris 27/07	-22.20	3230fc1808722e819051a4f5bb22fd70dad9838d3e344667b94b4f166e506aea	23	1	Service Navigo Paris	Service Navigo Paris	
602	14	2025-08-04	Carte Amazon Eu SARL Payli 02/08	-2129.00	08c9a7facba099b925d67b520ebab3fe79b074507689c9085b45c76093be8c2d	28	1	Amazon	Amazon	
579	14	2025-07-24	Carte X7303 Les Uli 23/07	-43.56	9e0de1233c12f39099f7ed9dcd3684e8c7fdd8fb4708788c0f40644aafa8cf8f	24	1	Carrefour les Ulis	Les Uli	
581	14	2025-07-27	Virement Vir Inst vers Trade Republic Pea	-45.00	12251f9515759fdef895373515694ba2efce5be3e6f5588c118297807d4b2133	29	1	Vers Trade Republic	Vers Trade Republic Pea	
537	13	2025-08-25	De M. Madi Toumani Ben-moh	115.00	9fc372b63c99191c56adca1fedc134f8ce46a23876e6193142bf30672fdc2b01	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
538	13	2025-08-25	Carte X7303 Burger King Les Ulis 22/08	-12.60	1a1952b7e14f8d23e541b85b2e99343ad3ab8d90b3befa83124ccfa62534705f	24	1	Burger King	Burger King	
513	12	2025-10-03	Carte X8028 H.t. Food Les Ulis 02/10	-8.80	2d08e13a1cace3a5de6fbf9383487aac8972bce8aa34a5158d08972ae8a15e3a	24	1	Les Ulis Food	Food Les Ulis	
592	14	2025-07-31	Web M. Madi Toumani Ben-moha	-2000.00	a8cb3e2936ab94ca2f323f0bfc49aa2b0bfdb7c07a297bab9c66ace568902763	31	1	Vers Livret A	Web M. Madi Toumani Ben-moha	
594	14	2025-08-01	Mens.pel 08/25	-50.00	9babab603aac51e1e78c5e861e2820386ff30c3f06452c3b9f729dd685e60bff	27	1	Versement PEL	PEL	savings
595	14	2025-08-01	Vir Inst vers M.madi Tarik	-50.00	02915847c761e064329da5e6fe59030ad760e2e862e5cd8a8e366a9a1b5542a5	29	1	Vers Tarik	Vers M.madi Tarik	
596	14	2025-08-02	De M. Madi Toumani Ben-moh	2129.00	517044bfb8fbdebf2dc7fceff744659645b33c7dfad5f613d420e69389eb891e	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
597	14	2025-08-02	De M. Madi Toumani Ben-moh	60.00	da9434653d887f4f92e110dd4e22eb2e7f43de94e9538617e39ecf5a2df11cd6	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
601	14	2025-08-04	X7303 Africa Exotique Les 02/08	-27.38	abd6b2c2d85bfc976aae866710fe1506a69245e078f1316555f4e7a49e23d69e	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
603	14	2025-08-04	Carte X7303 Burger King Les Ulis 31/07	-11.70	74f26b2e487a377be209f9077fadd2dd5980d924751e46cbefc4b06bcb686b57	24	1	Burger King	Burger King	
609	14	2025-08-06	Carte X7303 Burger King Les Ulis 04/08	-12.20	47918cec795dae553796f7ae3ce542454f428c9c6edfbe852c5fbc359a5b94f6	24	1	Burger King	Burger King	
611	14	2025-08-07	Prlv Acheel France REF-BED1D9B93DCAB4E1D3GFGWBH4 FR457Z787F911 Core	-11.44	94d07d2adadadcf6576446a8e182a0a35c87bbd8d67b23e41bff31de7d91463c	22	1	Assurance habitation Acheel France	Acheel France Core	
612	14	2025-08-08	De M. Madi Toumani Ben-moh	300.00	b8c4666aafef30a6b43e28057e5c3f077ea5bbfc8ff9378e7ead3cff7112448d	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
540	13	2025-08-26	Germanho Multi- Les 25/08	-6.39	faf99f7678baa3d459308c92bd8b94568aedc53d76ef4fb2ac379b8b08890751	24	1	Germanho Multi	Germanho Multi- Les	
583	14	2025-07-30	Carte Germanho Multi- Les 29/07	-6.35	a872354c0a19ae5335aaa477cdec0e8af684f0ddcd72897fb3172ec405595515	24	1	Germanho Multi	Germanho Multi- Les	
614	14	2025-08-09	De M. Madi Toumani Ben-moh	90.00	7aa5ce4de7cc56403c03c45be747a237682fd57361e1e0f25d8a6a48542d3005	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
615	14	2025-08-09	Prlv Offre Essentiel	-6.00	658c083350d387ec01150ef4e63ff004e2aad79df70931fb6a8a6a8d1ac32992	30	1	Frais Bancaire	Offre Essentiel	
584	14	2025-07-30	Carte X7303 Ulis Viandes Les Uli 29/07	-16.00	f54f9a4c1cd7c8b62b33ccea79bbf9877b532ab23542acecdb29b68d9f01722d	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
573	13	2025-09-15	Carte X7303 Ulis Viandes Les Uli 12/09	-17.61	f1d9e2faccc49aca2eee6550ba571bee989910cd0c9c7fb6024f722d8e822559	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
593	14	2025-08-01	X7303 Les Uli 31/07	-23.61	a058c4e92f455afebcfe0e0ac1724a69e3777c5c1c60aa3c6da5d184b328833e	24	1	Carrefour les Ulis	Les Uli	
607	14	2025-08-04	Ret DAB X7303 Les Ulis 2 Ga 02/08 12H32	-20.00	080c8a494e0001e80f99cba041f230d53ee5d3d6d9a2adc371cb50f9bfe73b80	24	1	Retrait Distributeur  Les Ulis	Ret Les Ulis 2 Ga	
605	14	2025-08-04	X7303 Les Uli 02/08	-16.09	19f24c977aca9c9244d5d76766c34be97eed482526e29ae55ba912e390b45a67	24	1	Carrefour les Ulis	Les Uli	
620	14	2025-08-13	Carte X7303 Ulis Viandes Les Uli 12/08	-33.00	642e1afca60e16eccf33b76c7a555be7c5faaa3c35e1cea908a6579b91766823	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
633	14	2025-08-20	Carte X7303 Nya*juste A Temps Pa 19/08	-0.50	6964138f9afcc70546ae282950e7360044629d6ae70e7464b7bee8cd593d42e7	28	1	Machine à café	Nya Juste A Temps Pa	
621	14	2025-08-13	Carte URSSAF Marseille 12/08	-288.00	03d812096db088d195322cfe11c27f0ae83e7dd3a36cb1045a4d467c1e0aae00	28	1	Cotisation URSSAF	URSSAF	tax
623	14	2025-08-14	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-13.99	dc1a4fa33f35f4f245a8350f5e52052c2719f947119170abf34e022bbb3a5eed	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
625	14	2025-08-16	Virement De M. Madi Toumani Ben-moh	250.00	5a05620d386aab5a48abc8ea21b15d4a9c858625fcb1e46f8384710c68d231b4	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
627	14	2025-08-18	De M. Madi Toumani Ben-moh	50.00	2e6648a099c6c8ab0c2872b3b276f00935725bf16e9cee9d00e2951616989285	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
629	14	2025-08-19	Carte X7303 Nya*juste Temps Pa 18/08	-0.50	8b673a2f6a80f344f0a165d201787e23e64a3721a189815004543a377176a71c	28	1	Machine à café	Nya Juste Temps Pa	
630	14	2025-08-19	Prlv Engie Mandat Core	-25.65	e1e03cffdd4049f2b5deacd49900da20c517bf45d2a69d1ab69b88eccd809bef	22	1	Engie	Engie	
632	14	2025-08-20	Carte X7303 Nya*juste Temps Pa 19/08	-0.50	153a3ecdc71878ede3c94d218f3e7fe4617b7072c0d03ba6f5ff9cc75ac0fb52	28	1	Machine à café	Nya Juste Temps Pa	
628	14	2025-08-19	X7303 Les Uli 18/08	-10.52	0930593317e0b9261fe7b0d5ac00490ad0d5e81c5c8889a1812aba9c4aeb96de	24	1	Carrefour les Ulis	Les Uli	
635	14	2025-08-21	De M. Madi Toumani Ben-moh	25.00	39f59e26d1bb1a74e818e7f8261e53e094f207db22578f851190e24f97f752ec	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
636	14	2025-08-21	De M. Madi Toumani Ben-moh	70.00	76a20dee909ed886a089ee3bb8d72a844d872f5a01e3cb427831ce32659e20b5	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
637	14	2025-08-21	Carte Nya*juste Temps Pa 20/08	-0.50	e9f96fb365a998febe1129c9964a1ffcefa6b36b4efd5ed96260dc18f0aa0910	28	1	Machine à café	Nya Juste Temps Pa	
639	15	2025-06-21	De M. Madi Toumani Ben-moh	100.00	f3960ff3938b76ceeb9fc51146bbac511e48c1e62499e55c0ad1685c9eedb28f	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
640	15	2025-06-21	De M. Madi Toumani Ben-moh	50.00	e084f2631922f06b28205e4ff17e4ba17086364f94c773ca83397ac2a78f14b7	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
587	14	2025-07-31	Virement Lessonsharing MTB-2025-31-07F:1-16-05-1	840.00	62aa494997eea0112e7da38c767472f6efad304fc272784de156bba01737e95e	28	1	LessonSharing	Lessonsharing - - -1	
588	14	2025-07-31	Lessonsharing MTB-2025-31-07F:1-06-06-1	840.00	e52cf94a287cde892ee8b17d79e5fbf7910341df4b14f4e08b9cb18debfe091d	28	1	LessonSharing	Lessonsharing - - -1	
591	14	2025-07-31	Web M. Madi Toumani Ben-moha	-4000.00	94e029c0fa5c290cf1785d96e2012a85499088bb66355c291e32408aea60be8c	31	1	Vers Livret A	Web M. Madi Toumani Ben-moha	
648	15	2025-06-23	Carte URSSAF Marseille 20/06	-521.00	9772015c8bad31a7ae412c64ea74011689b33093344f7462f25924aad3e0ee81	28	1	Cotisation URSSAF	URSSAF	tax
649	15	2025-06-24	De M. Madi Toumani Ben-moh	280.00	f2836e1c455843b353f71fa130eaddba0ee5ed9627b3da6a87f1b5215349891f	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
650	15	2025-06-24	Carte X7303 Amazon Payments Payl 21/06	-89.99	11ac247a216dcfb1a6b35677803efa0492076c5f894eaf86d7257ed36d1b93a0	26	1	Amazon	Amazon	
653	15	2025-06-25	Carte X7303 Burger King Les Ulis 24/06	-12.60	519ccd9c62ccb025ef4b83515a97d0ba4fd6befdea1b864d66c78b0c1b2dec73	24	1	Burger King	Burger King	
654	15	2025-06-21	De M. Madi Toumani Ben-moh	130.00	16288e055217ec2cf095f72389f3a928c5275f5ae18ee448eb465615d7f9bb4b	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
656	15	2025-06-28	De M. Madi Toumani Ben-moh	100.00	8481c74a5c7147809c45639b3f85fc14992794835886cb55fc55edab8eb1c867	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
647	15	2025-06-23	X7303 Les Uli 21/06	-30.65	54e43e9ea9ca64ef567607dbc1875cdcdefc710ad83ce0c18094da25053b9acf	24	1	Carrefour les Ulis	Les Uli	
645	15	2025-06-23	Carte X7303 Cleor Les Ulis 21/06	-44.50	15d16e88d9092731d05777eb090e81181b639ab8e6081a8d24bc4f0713c0703f	24	1	Cleor	Cleor Les Ulis	
618	14	2025-08-11	Vir Inst vers Trade Republic Pea	-250.00	ca7e72f2b90cbc65298f8d43692666f51ae2249f3488f60704b788b73eeeebcb	29	1	Vers Trade Republic	Vers Trade Republic Pea	
660	15	2025-06-30	X7303 Les Uli 28/06	-34.77	fd985facfbb63ea79f3dbcc5b78d997b99761361928383f948c6f052ea2878cd	24	1	Carrefour les Ulis	Les Uli	
604	14	2025-08-04	X/303 Maison Nouilles Pari 01/08	-17.40	1aed328e0041508e5351ed357232dd70ff1b97204f5083a0f39c03278806b72b	24	1	Nouilles Paris	X Maison Nouilles Pari	
662	15	2025-07-01	Virement Mens.pel 44690429796 07/25	-50.00	a7554920ec047ccb673536316f238b3e939e5101a17eac8faeb70e0b1358fee5	27	1	Versement PEL	PEL	savings
663	15	2025-07-02	De M. Madi Toumani Ben-moh	600.00	f3bae91a0321891ba6cb17a28152e5e17e1eece38b08b344531df8070f69af80	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
668	15	2025-07-03	Vir Inst vers Mme Jing An	-2850.00	b3170dbd11b0ac1b1809f45109c07996f4dd06b9cab8170d506bd884c60399ed	22	1	Loyer	Vers Mme Jing An	
669	15	2025-07-04	De M. Madi Toumani Ben-moh	500.00	178b083abdad4deddc35a498b2a7352bb7d93c3dcf7aa74c633bc17ac6153da9	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
670	15	2025-07-04	Prlv Free Telecom Free Hautdebit 139630085347939382 FR83277459654 Core	-19.99	c13e30a3ea1744cf3ea537344fcb0d84b09d57767efa5b311067698310b135be	22	1	Free	Free	
671	15	2025-07-04	Vir Inst vers M.madi Tarik	-280.00	6bcaa9d6d9be15b36314b4e28a8eadb21302786196c6bedcb6523198726d7adb	29	1	Vers Tarik	Vers M.madi Tarik	
619	14	2025-08-12	Virement De M. Madi Toumani Ben-moh	300.00	b90e9615bb78202073f0c44d3b3d63130fde52a9232ec19bad1163221fa7eb9a	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
675	15	2025-07-07	Virement De M. Madi Toumani Ben-moh	150.00	e95b3c95395101f1854de6e5c589b5f65e11c8f00b1f85788a6470fdbecacc97	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
676	15	2025-07-07	Carte X7303 Amazon Payments Payl 02/07	-27.28	d99aef71a3faeaeffc8276a96800605e34dff23769b718486b37ca5029e0088e	26	1	Amazon	Amazon	
680	15	2025-07-08	Prlv Acheel France REF-A56561E3996BC21E94GFGWBH4 Core	-11.44	020df0815c68683897b8e0b6371daebb199c54fc3ccb895758b4794d1985194e	22	1	Assurance habitation Acheel France	Acheel France Core	
681	15	2025-07-09	Prlv Offre Essentiel	-6.00	69a584e2a8ebc95a50c31de2fc2c46a28432f1a876d249b74d323a03ead58cf5	30	1	Frais Bancaire	Offre Essentiel	
658	15	2025-06-28	Virement Vir Inst vers Trade Republic Pea	-28.00	0e6bbb94458faa8ea8fa1243edc30179ea7e6e1fa716dd85fb139111576f3ac4	26	1	Vers Trade Republic	Vers Trade Republic Pea	
684	15	2025-07-11	Virement Vir Inst vers M.madi Tarik	-50.00	a328b214772b941133a399708c240875f0daa13ce46597bb04b647e7ddc23e26	29	1	Vers Tarik	Vers M.madi Tarik	
685	15	2025-07-15	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-11.99	6e4a0b7d8c4eaf9b9da0980fbca9ce6ecfac58bc58e51e82e7901d99c713a019	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
686	15	2025-07-16	Virement De M. Madi Toumani Ben-moh	250.00	465f1277d0b4d9aaef1dd0e27d28841b2a68c3885081bc6ecda38bccf7ec81eb	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
687	15	2025-07-17	Prlv Engie Mandat Core	-57.00	f87c485f8d5829f0795cd6e809d753962621a6d9c8c4af75fbd7ddbc9798e689	22	1	Engie	Engie	
689	15	2025-07-19	Virement De M. Madi Toumani Ben-moh	200.00	4bc178c4170e28fba761df734a54ce6b3138335a84ac77b341ba1ab61c5de7b1	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
691	15	2025-07-21	Prlv Paypal (europe) $.a r.l. et Cie, 1043587480928 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-222.09	396938dedee7e8966a6b28db943eb69ef8bee2afc1eb7989642851b7113e6c96	26	1	Paypal	PayPal	
692	16	2025-05-22	De M. Madi Toumani Ben-moh	300.00	eeccfa16eb14a3937acaf63dc7525131dc86fdd5e4eb0dabcf4b6abe232bd2ec	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
693	16	2025-05-22	Virement De M. Madi Toumani Ben-moh	50.00	f012086a0be70c8ee60a5f798e4213e579024c66eff37cc04f35166f5583e460	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
694	16	2025-05-22	De M. Madi Toumani Ben-moh	710.00	f5b0c996511a6f13531fe04445cb5cb44dd30964b160c66c0fd585fa23c2cf58	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
705	16	2025-05-26	Carte X5/81 Photomaton Echirolle 22/05	-10.00	e051e72907c63638e98a078d0a5aec4043e10778a5a99050cb965deaeda0557f	30	1	Photomaton Echirolle	X5 Photomaton Echirolle	
664	15	2025-07-02	Vir Inst vers Ericia Arielle Rev	-500.00	6fb89242b2c5b6c7ca723ef19f9edb10b518aaa8797ee2286e97a8fad8ef9adc	30	1	Vers Arielle	Vers Ericia Arielle Rev	
707	16	2025-05-27	Virement Vir Inst vers assjad	-4.00	2297a081c7b455e612864c99e1bf9c62db66770c36c457556581e79398376eb0	29	1	Vers Assjad	Vers Assjad	
642	15	2025-06-21	De M. Madi Toumani Ben-moh	120.00	5aee0f2ea7d23da275a8af022593040efd05c14326fd9c529cb42f198cd2ee93	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
643	15	2025-06-21	Vir Inst vers M.madi Tarik	-100.00	17848abc16cfb778419697fb10c63de70cfb6a6dbd030e0faa4f65ea72b77abd	29	1	Vers Tarik	Vers M.madi Tarik	
646	15	2025-06-23	Carte X7303 Deliveroo 75009 Pari 22/06	-24.58	561921da33de26b0ed23b0077102c35d4772d8bf83e3139f6cb9b04125f9ab93	24	1	Deliveroo	Deliveroo	
674	15	2025-07-07	Virement De M. Madi Toumani Ben-moh	200.00	d019c0cbf8799de9cbd7eda96af03f43833a1a9bba1855fb8d1e5cb813547a54	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
735	16	2025-06-10	Carte X7303 Ulis Viandes Les Uli 06/06	-52.15	140e34fbc27c8ceb9a18d517c5479aa94233d09bf566a713d6060675c332068f	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
704	16	2025-05-26	Carte X5781 Deliveroo 75009 Pari 24/05	-15.03	43a9f42f8255e31dbc4fd61464d1e243b9372a47061e01d287cf41f4761d2435	24	1	Deliveroo	Deliveroo	
708	16	2025-05-30	Virement Lessonsharing MTB-2025-05-30F:1-12-04-1	1440.00	3c0a38ab23d07cb0b2bf0c59eee3f07e18e3d04924a47300be53d4b1ffd6150d	28	1	LessonSharing	Lessonsharing - - -1	
709	16	2025-05-30	Virement Lessonsharing MTB-2025-05-30F:1-05-04-2	350.00	55e6c16bf83d6fb00018a55dd3a17baa619d6d19b48fc75fc781f8baa7210fbb	28	1	Lessonsharing	Lessonsharing - - -2	
710	16	2025-05-30	Virement Lessonsharing MTB-2025-05-30F:1-05-04-1	690.00	0c89024fe355651773338387b78770a0bd3096104481b769978633f75813ef17	28	1	LessonSharing	Lessonsharing - - -1	
682	15	2025-07-09	Carte X7303 Caresquick Zaventem 08/07	-13.65	1bae11f094c16790238bfe859b0b4ce77f9d66075f28da275b1dbd876082103a	24	1	Caresquick	Caresquick Zaventem	
712	16	2025-05-30	Virement Lessonsharing MTB-2025-05-30F:1-01-04-1	230.00	838f2e86339e57846301c1188cf7a46df06c7e5c4e83af84268fb1f62ed90d85	28	1	LessonSharing	Lessonsharing - - -1	
713	16	2025-05-31	Vir Inst vers M.madi Tarik	-100.00	3bcda5f452a9fafe5a82f3eb4635ddbb4b6918201e2aa1dc399d2dcc4fbadf59	29	1	Vers Tarik	Vers M.madi Tarik	
717	16	2025-05-31	Web M. Madi Toumani Ben-moha	-2600.00	b3940b0c0a26e328704adfe6531a98010cced5f3fbe599fea6ffad4e04d3c72b	31	1	Vers Livret A	Web M. Madi Toumani Ben-moha	
718	16	2025-06-01	Mens.pel 06/25	-50.00	3a800b3d4b13e8518da93c70a3f206559e9ec08d68117a5709d40a917b150fc4	27	1	Versement PEL	PEL	savings
702	16	2025-05-23	Ret DAB X5781 Les Ulis 22/05 12H11	-300.00	d1798fad394b53a3f922b7afc4c578d729b533637b93f322d956ddb433e54c25	24	1	Retrait Distributeur Les Ulis	Ret Les Ulis	
720	16	2025-06-02	X5781 Service Navigo Paris 31/05	-22.20	bc95b4a9265ebeb3c3d9c4c167436653f10150834e1ae16a653e42c5508922e5	23	1	Service Navigo Paris	Service Navigo Paris	
729	16	2025-06-09	De M. Madi Toumani Ben-moh	385.00	2793e8fdac16984eaac05b04a8c48af899ff3e206eb9a88f14723e54bf769ada	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
730	16	2025-06-09	Prlv Offre Essentiel	-3.00	24ba76f472d5a7ea34c34bcae0e346146e51422b3fc54fbaeaf4cd19da116638	30	1	Frais Bancaire	Offre Essentiel	
732	16	2025-06-10	De M. Madi Toumani Ben-moh	300.00	693f4754fea12a400ba68ed8c7584520641a4e4488e222e7f67a2a30b0f97691	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
733	16	2025-06-10	Carte X7303 Africa Exotique Les 06/06	-17.97	24cfc812294cec7778adb8cd5cddc0d2ab70c0f2a89450596b067ddfcbb03c0a	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
703	16	2025-05-23	Ret DAB X5781 Les Ulis 22/05 12H13	-50.00	6675bf66751d8974e7d99702d639029769719e6a68c44ae3521afd7e31052e47	24	1	Retrait Distributeur Les Ulis	Ret Les Ulis	
736	16	2025-06-11	Carte X7303 Deliveroo 75009 Pari 07/06	-19.01	9f1572ebc6f558c4e32d3536a4cdff841efb6318912a9d9bccaba55923712e31	24	1	Deliveroo	Deliveroo	
738	16	2025-06-13	X/303 Temu.com Dublin 2 12/06	-24.93	b4c1fa9c78a48690716e04698803002029293214d9da44eb5c45cd737d9ce697	26	1	Temu	X Temu.com Dublin 2	
706	16	2025-05-27	Carte X5781 Les Uli 24/05	-39.45	8a172e6185ba73de3964675d7472653e2daffa0fac8e7781b3ce3baca48aa820	24	1	Carrefour les Ulis	Les Uli	
734	16	2025-06-10	X7303 Les Uli 06/06	-46.50	1682dadaa937bb6078b0a7f15ae858eeba0a2eb1524e3feadd54740adb7db405	24	1	Carrefour les Ulis	Les Uli	
758	17	2025-03-30	Virement Vir Inst vers assjad	-15.00	6c5071bba313ac1d3baf0dce07b4d7f0fa6e2ca440d8d938f146956a74253900	29	1	Vers Assjad	Vers Assjad	
721	16	2025-06-02	Virement Vir Inst vers Arielle Rev	-623.00	a8b7c4860fb920a35ce116708b9aa9812c818db0138a9cd6a476d388980128db	30	1	Vers Arielle	Vers Arielle Rev	
726	16	2025-06-05	Prlv Free Telecom Free Hautdebit 138873464347939382 FR83277459654 Core	-19.99	8118fa259e1e53434f48695a58afd62b6d4ce80829b3e1df2d0f546e9790f7f8	22	1	Free	Free	
747	17	2025-03-24	Carte X5781 Timbre Fiscal 35 Ren 21/03	-30.00	212f7f88a0a90e6416005af885ea6171f4aaeca3403aa7d1542d6c82a4c42c8b	29	1	Timbre Fiscal	Timbre Fiscal Ren	
770	17	2025-04-03	Prlv Intérets débiteurs Période 1 Trim 25 TAEG=22,47%	-0.02	183f3d628438cba9fbf463bb42e0777bd56b7cbaab87e8a327205087efd68a9d	30	1	Intérets Débiteurs	Intérets Débiteurs Période 1 Trim Taeg= , %	
761	17	2025-03-31	X5/81 Chatelet Resto Paris 28/03	-4.70	a78831d7a197231bcd6c9bd7d3573cdd1edec3de81905a98e3dbd9514d566b85	28	1	Chatelet Resto Paris	X5 Chatelet Resto Paris	
741	16	2025-06-16	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-11.99	0d764c773454088d3c26e58151ccee8ed675eeeb986ab6e618034523d9501a24	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
742	16	2025-06-17	Virement De M. Madi Toumani Ben-moh	100.00	ebceca74b084214b18e1967e8071b59ed6a2aa594acc0d9a180e20fb6e8a0e7d	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
743	16	2025-06-17	Prlv Engie Mandat Core	-57.00	44c673313fbc27da37f037836a3077026875982696dc5afc031b9553a673736e	22	1	Engie	Engie	
744	16	2025-06-19	Prlv Paypal (europe) $.a r.l. et Cie, 1042938270102 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-222.09	e12ea4f7f4ea9de4bd33a86ed520946f1f89381db7c23fd7ff102533f0626d8b	26	1	Paypal	PayPal	
745	16	2025-06-20	De M. Madi Toumani Ben-moh	530.00	b82bc2751e3a0f24167c54e2f334be6d2b1634b480313f1e58e7f9ed9832da08	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
325	8	2026-06-30	De M. Madi Toumani Ben-moh	100.00	29c8ab8695ffde7dc6073bf539ee269692a27db7432af3ee685f4e26862abd36	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
696	16	2025-05-23	Carte X5781 Lp Photocopieurs Lyo 22/05	-1.60	53b87a260e509ca4600ec7ffad0f0cab3c0d02e2114939897e6c060e0be57322	30	1	Photocopieurs	Lp Photocopieurs Lyo	
697	16	2025-05-23	Carte X5781 Lp Photocopieurs Lyo 22/05	-0.40	f42ac045a6b155dd89930a81ddc2feb61f6bdbd64ab2202cd87a28665ac918e6	30	1	Photocopieurs	Lp Photocopieurs Lyo	
719	16	2025-06-02	Lessonsharing Facture 1-19-04-2	420.00	c5886c69310c35fba7760b056b8e6dd612198cdea93b36099d667ec8ffc2c649	28	1	Lessonsharing	Lessonsharing Facture 1- - -2	
739	16	2025-06-15	Vir Inst vers Trade Republic Pea	-25.00	ed38d8ccf53623113d393d5a2af6a2a201072b6cd76734d177baa542803cf9cf	44	1	Vers Trade Republic	Vers Trade Republic Pea	
757	17	2025-03-29	De M. Madi Toumani Ben-moh	50.00	8f24a266517c5b3fd73265b7027026c5cfcdf9a83d3ea74b01f7107f1547dd49	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
759	17	2025-03-31	De M. Madi Toumani Ben-moh	100.00	3cb898a461330a68b78c8d609a919622e969483d4dae289998f4edf5005096e2	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
760	17	2025-03-31	X5781 Africa Exotique Les 29/03	-10.78	f189b16c20085750e88891f433f265dbdd668ba78d706a6c3f54e5356519c070	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
724	16	2025-06-03	Carte X5781 Coinbase Dublin 02/06	-50.00	1fc316cacd427717d70136e3c56781cad1b2bc51c7a9f9bba2e06657473c66bf	44	1	Coinbase	Coinbase Dublin	
762	17	2025-03-31	X5781 Service Navigo Paris 28/03	-22.20	f3c077d78a0ee150321fc09b24731c21d6eee137bdff3edbdb460d3ea21c463c	23	1	Service Navigo Paris	Service Navigo Paris	
750	17	2025-03-26	De M. Madi Toumani	650.00	1120af99e6d22632cfdb5ed7389d99fcfd0feb7adc8761f2796bd2fc7413cfd3	31	1	Vers Compte courant	De M. Madi Toumani	
765	17	2025-04-01	Carte X5781 Ubr”* Pending.uber.co 31/03	-82.94	08cc19b95c2614bb0f39d9fc5f024cd2d71b1f567c506ae7166d8ed2be1243ab	23	1	Uber	Uber	
767	17	2025-04-01	Mens.pel 44690429796 04/25	-50.00	ada5079e461bc8c0a8c23d32fccb3180cc5005fe1db6cd4574dee8abb0cc677c	27	1	Versement PEL	PEL	savings
771	17	2025-04-03	Carte X5781 Burger King Les Ulis 02/04	-12.10	c5ec5ec7cd68033d481a8b6a6cb7c6249c01db8ca5d34e5173d61fed64ed118b	24	1	Burger King	Burger King	
772	17	2025-04-03	Carte X5781 Nya*juste Temps Pa 02/04	-0.50	78d23d42adb52cfca6d1ad9cdf94bc99e6a4d0090afa959fcec64cb39c88d902	28	1	Machine à café	Nya Juste Temps Pa	
773	17	2025-04-03	Carte X5781 URSSAF Paca Ce Marse 02/04	-387.00	da62ef857c622548b8fb13ab9723cc663a4cf91080ebc88420aa1756402f8e03	28	1	Cotisation URSSAF	URSSAF	tax
806	17	2025-04-17	Carte X5781 U Express 69 Lyon 16/04	-9.54	799766aa5b8af1d77e83502867e89ffbd1cd752766bf417ac3c25a6f247c0d81	24	1	U Express	U Express Lyon	
764	17	2025-04-01	Lessonsharing MTB-2025-03-31	3100.00	1119db5703f837a7fcb0b0a5eee024ed24f5416619b1478bc9c30447f6bf26a5	28	1	Lessonsharing	Lessonsharing - -	
737	16	2025-06-12	Carte X7303 Les Uli 11/06	-9.20	be0aa89573afb43447d34077e7cc4d249fbb905d3e26d6cbf8dcf4b3f033c75e	24	1	Carrefour les Ulis	Les Uli	
756	17	2025-03-28	X5781 Les Uli 27/03	-31.00	ea00a312f56f81a78974e289bcb9e95e0902b30ed62e3fa5451651e45aaed3d0	24	1	Carrefour les Ulis	Les Uli	
776	17	2025-04-04	Carte X5781 Nya*juste Temps Pa 03/04	-0.50	9a0a578fb94530317620264b443d6d0819dad3c40efd51b747bedf0c78d816dc	28	1	Machine à café	Nya Juste Temps Pa	
786	17	2025-04-09	Carte X5/81 Pizza Andiamo Les Ul 08/04	-15.00	16776ae2d5512266eba8d90a08343dbb2d699587cc03c4aa36940b12400f75e0	24	1	Pizza Andiamo Les Ulis	X5 Pizza Andiamo Les Ul	
775	17	2025-04-04	De M. Madi Toumani Ben-moh	330.00	aa25cf49b3569c22cc71e722607a5f020c654cf52d6306696e1f509e176303f8	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
782	17	2025-04-08	X5781 Les Uli 05/04	-36.36	42e3e8762941b5f409a27f46e5321cc8ba67f0c3321e86184cca206f72043037	24	1	Carrefour les Ulis	Les Uli	
777	17	2025-04-04	Prlv Free Telecom Free Hautdebit 137354377847939382 FR83277459654 Core	-19.99	972f36d592fe5c5636ae986f5514c39b76481daac73888485f6b4b8347d352d5	22	1	Free	Free	
778	17	2025-04-04	Virement Vir Inst vers Mme Jing An	-2850.00	cefba47f882d7f571ef4e07fb63d436624f3691c7d9fadb224b497f5982a1fa6	22	1	Loyer	Vers Mme Jing An	
780	17	2025-04-08	De M. Madi Toumani Ben-moh	70.00	ffca1411f6039fde4134dec9c819887e2ac30c2d02c911882e88d7549f7e131d	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
784	17	2025-04-09	Prlv Offre Essentiel	-3.00	7433095dab61d0ccc7c17f4b27f5689f89fc0acc508d4a03717f476c24a7eaea	30	1	Frais Bancaire	Offre Essentiel	
785	17	2025-04-09	Carte X5781 Africa Exotique Les 08/04	-12.66	da6ab681c17a2f98c10e8bfa1ba2fe860bf3d24e21a00a6ee844a6186bd2d8e7	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
787	17	2025-04-10	De M. Madi Toumani Ben-moh	400.00	2813cbfc75c6b3b50f4a119bb27a63cfd3ddef7341a497a48769070756b844e9	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
789	17	2025-04-11	De M. Madi Toumani Ben-moh	200.00	2f5cce9142cd55cbbe1095ea8458d5b64130d43e2f0a4a5405ec1c8f81a5c7a7	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
791	17	2025-04-11	Carte X5781 Sncf-voyageurs Paris 10/04	-89.80	1ddbece7dcaf90b0d60f8dead2b694708505ae4f748d07e01595082f02305897	23	1	SNCF	SNCF	
793	17	2025-04-11	Vir Inst vers M.madi Tarik	-200.00	a034d95231d125470da8592270243743a091f51aa2b772376586d9b993ea00cd	29	1	Vers Tarik	Vers M.madi Tarik	
794	17	2025-04-12	De M. Madi Toumani Ben-moh	50.00	6b85d04f20087670c5f2f71fa70a6e817317f9ed1b5bc42d6a716446bb1bde6b	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
796	17	2025-04-14	De M. Madi Toumani Ben-moh	20.00	ae351f3c73f3c932b612109d056f4035738d6d846dd637025f2468f07eff288e	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
797	17	2025-04-14	De M. Madi Toumani Ben-moh	30.00	a416935c1ce49456b2d3e95d05f4db36ff15f83b6af8d2e23f1c6aed0d18b6d5	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
802	17	2025-04-14	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-11.99	b8eee818b0dde996f669112418dce305884aac7eb1ee6add6925ae84d4006a8b	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
769	17	2025-04-02	Carte X5781 Nya*juste Temps Pa 01/04	-0.50	f9337d4d22863335f0f54e9fd7687127377e4943a612844e5c8bd4d5d8e21c2b	28	1	Machine à café	Nya Juste Temps Pa	
803	17	2025-04-15	De M. Madi Toumani Ben-moh	50.00	c8840944ea37c17602c233a3920b2796fe397b21336ff25535d69eec83f6143f	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
749	17	2025-03-25	Prlv Paypal Europe $S.a-r.l. et Cie S. 1041071344233/PAYPAL567J2253W2NHA LU967ZZZ0000000000000000058 Core	-9.75	84e2430e067807fa98b15b6cc0c310ed36a20689beef82c608f9416a8f04c82e	26	1	Paypal	PayPal	
755	17	2025-03-28	Carte X5781 Amazon Eu SARL Payli 26/03	-19.90	5b5f8f66fd2d6ded1ed98bff2dac73eeac20dc5597ad84213193b538b631fa5d	26	1	Amazon	Amazon	
752	17	2025-03-21	De M. Madi Toumani Ben-moh	100.00	851bf1fff747f8ca1a660e55198ad0b9ec9958573678d807dfb71121de0c8394	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
812	17	2025-04-21	De M. Madi Toumani Ben-moh	350.00	11ae2f9f39013877800d17fcf4918d0593f2bb19e9e2e1947018573a1dbb0e74	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
774	17	2025-04-04	CAF De L Essonne 37655511V167990165TMADI TOUMANI032025ME	39.00	f246bde8f7f350acb1145bf8554d9c6500c70c0dbed60a8dd0e31b33092ab56d	30	1	Prestation CAF	CAF	benefit
809	17	2025-04-18	Carte X5781 Temu.com Dublin 2 17/04	-28.02	a7cd4c74723a8035da8f1e53fd32967d08a6bdf60fd21f8e863d016bf0258a2c	26	1	Temu	Temu.com Dublin 2	
801	17	2025-04-14	Carte X5781 Smile World Malakoff 12/04	-11.00	e4e50660e124a2be517664b41471e923971ca66b2e4623eecc18dfa1aa894308	33	1	Smile World	Smile World Malakoff	
821	18	2025-02-24	Carte X5781 Pharmacie Ulis 2 Les 22/02	-19.29	90d8207a182e009584c806fe0a2c212864a1602b2d896ddcb41ef55fe5107ed9	25	1	Pharmacie Les Ulis	Pharmacie Ulis 2 Les	
838	18	2025-03-05	CAF De L Essonne TOUMANI022025ME	39.00	18283ead310f31222bfcc8cbce416c31156d4da9c186327ab8deed4edac78ccd	30	1	Prestation CAF	CAF	benefit
819	18	2025-02-24	Carte X5781 Instant Gaming Dubli 21/02	-2.31	9621ef90bb8f9a9af567dc59b3ff815ee9058e67fd2265787c318c85e215ab0b	26	1	Instant Gaming	Instant Gaming Dubli	
839	18	2025-03-05	Carte X5781 Coinbase Dublin 05/03	-19.99	fe5cb7ad46da6235895f6ad7c35da398c09f0b23418d977a93b69671c4d7c368	44	1	Coinbase	Coinbase Dublin	
830	18	2025-03-03	Virement Lessonsharing MTB-2025-02-28	5060.00	cdf75751564c2072409b3f370d0b549ed2f9b6f29cbdff0198725e66ba95958b	28	1	Lessonsharing	Lessonsharing - -	
827	18	2025-02-28	Virement Vir Inst vers assjad	-12.00	55d42c3aea20996ef7beffdcb1a6276f30113c9015211b397aa74e7a939a437b	29	1	Vers Assjad	Vers Assjad	
816	18	2025-02-22	Virement De Madi Toumani	500.00	773c89046cdb5cb6641cd53a4441b1513635bdda6018e3765f0984f5564731c8	31	1	Vers Compte-courant	De Madi Toumani	
817	18	2025-02-23	Vir Inst vers Arielle Rev	-350.00	cbc514a538567dae25291b360c3d788c5af407e272dc3742214d8dcb3c6e60e8	30	1	Vers Arielle	Vers Arielle Rev	
814	17	2025-04-22	Carte X5781 Les Uli 18/04	-57.65	b6ced0415e872bab4d7680458777298f1230c3457f44c2cbf30edd8d634edc97	24	1	Carrefour les Ulis	Les Uli	
856	18	2025-03-10	Chamas Tacos Marseil 08/03	-13.40	c239b243494a9ec3ddb05c307b965dc2e341e344a32d222b0fdcc5a515f8ca43	24	1	Chamas Tacos	Chamas Tacos Marseil	
818	18	2025-02-24	Carte X5781 Africa Exotique Les 22/02	-16.95	169cafe5882a4ca8a41a1cde687b9be8edd6f0730100c35ad048622f091b764f	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
820	18	2025-02-24	Carte X5781 Les Uli 22/02	-32.44	ba26f15bd9de16d8441fc81bcb8f5f9ab6d680e92977029b2144a19f4dc0f168	24	1	Carrefour les Ulis	Les Uli	
842	18	2025-03-06	X57/81 Coinbase Ireland Dub 05/03	-50.00	cedc7557f7dd7613848bdf74ac90744ebce7aa0aea3792bd8f01225c75f91335	44	1	Coinbase	Coinbase Ireland Dub	
835	18	2025-03-04	Carte X5781 Megdiche Marseille 03/03	-17.80	6b1d9b38f5d7998db80c0e517eb71104932a05f7e2f03278b454c86c1b31c57c	24	1	Megdiche Marseille	Megdiche Marseille	
826	18	2025-02-28	Carte X5781 Service Navigo Paris 27/02	-22.20	9dda6650cafdc8e1c6bbf73a0b88e568670ed46af0f6501a77b2f2e04b356406	23	1	Service Navigo Paris	Service Navigo Paris	
829	18	2025-03-01	Virement Mens.pel 44690429796 03/25	-50.00	c9820789e10be69146ba1125c05ca3ea3425f3c184c48ab710d15c2eb413e638	27	1	Versement PEL	PEL	savings
823	18	2025-02-27	Ret DAB X5781 Paris Chateau 26/02 16H57	-20.00	0ac5f90eb47bfbb6442c0719fe7cc3298bd2de5eb94e836fffd3c50f3d8745d1	30	1	Retrait Distributeur	Ret Paris Chateau	
832	18	2025-03-03	Prlv Paypal Europe $S.a-r.l. et Cie S. 1040563399226/PAYPAL567J2253W2NHA LU967ZZZ0000000000000000058 Core	-13.49	1754e90ba8aa758e68ee6595456f4e0a2ac130dbb247032b523eecf178f1893f	26	1	Paypal	PayPal	
815	17	2025-04-22	Carte X5781 Ulis Viandes Les Uli 19/04	-12.09	c8e6ff01173a72da78626c07bc0875a0b254c9361683050df4365ff95d1bb2fe	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
836	18	2025-03-04	Carte X5781 URSSAF Paca Ce Marse 03/03	-632.00	214ef2189fcf20fb1e277ce27afcfb5a32b6fb055d377a209c6d2f757420ed01	28	1	Cotisation URSSAF	URSSAF	tax
840	18	2025-03-05	Carte X5781 Sncf-voyageurs Paris 04/03	-51.00	757f1f6a34b5c2cc542f0fb640dcc59c6f17626999a245821aaff6ab978ddaab	23	1	SNCF	SNCF	
841	18	2025-03-05	Prlv Acheel France REF-893671E654A63D1ED0GFGWBH4 Core	-11.44	c1e1c06ff3186eb982c71dad96850bd6175fa9a1e4890543a46e3647da67555c	22	1	Assurance habitation Acheel France	Acheel France Core	
847	18	2025-03-06	Carte Sncf-voyageurs Paris 05/03	-59.00	d6633382127bae7f54eb86f0483ddf0a8971731799954a9f19c032b7fe0b626e	23	1	SNCF	SNCF	
848	18	2025-03-06	Prlv Free Telecom Free Hautdebit 136595844247939382 FR83277459654 Core	-19.99	cbeb8b59fa93dc4f3ba334a258db72c82997cfce77e51704a91e725503d7735d	22	1	Free	Free	
843	18	2025-03-06	X5781 Coinbase Ireland Dub 05/03	-50.00	85791495dd07a0655f98dbd784f5bec5d98eca0d83dc3ec8ece2aac575159557	44	1	Coinbase	Coinbase Ireland Dub	
851	18	2025-03-07	Carte X5781 Amazon Payments Payl 05/03	-189.99	62ff5c935542c62fbdebcbdd2f31aa0aed62ac026f8aa94deb9e687299172031	26	1	Amazon	Amazon	
852	18	2025-03-09	Prlv Offre Essentiel	-3.00	2c58f152700f61c7f28d2230210b71c9c4026505164111216f198afd14cea216	30	1	Frais Bancaire	Offre Essentiel	
853	18	2025-03-10	Paypal Europe S.a-r.l. et Cie YYW1040780956821YYW1040780956821/PAYPAL	31.76	ac9be163b2aab783b6e103399dab2dcf8f4a547ed69b2ceea30eb840813d7101	26	1	Paypal	PayPal	
854	18	2025-03-10	Paypal Europe S.a-r.l. et YYW1040740346361YYW1040740346361/PAYPAL	16.00	bed6030da3a4443bc84daf4d81128d2a9dc60477805f050fde6f2896f2332d20	26	1	Paypal	PayPal	
837	18	2025-03-04	Virement Vir Inst vers Lessonsharing	-5060.00	bc7b0c7523d71bc228acc7097b9c91a28c58343a9beb4e3202fea5c25173b30f	28	1	Vers Lessonsharing	Vers Lessonsharing	
781	17	2025-04-08	Carte X5781 Lh Boucherie Les Uli 04/04	-11.35	6d2c954c2d25b0b8c027bcc07a07b8de934c0205357cc7ef0cf24079cc1600e6	24	1	Boucherie Les Ulis	Lh Boucherie Les Uli	
807	17	2025-04-17	Prlv Engie Mandat Core	-57.00	c1ed718edcab76920e521186f11027cc5129b0286ef924e5b6f70f9243803b6d	22	1	Engie	Engie	
808	17	2025-04-18	De M. Madi Toumani Ben-moh	130.00	6329b0286097306b0fd540c048c431dc1ba94346c54cb8e5df731db897fb27b0	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
863	18	2025-03-11	Carte Amazon Payments Payl 08/03	-220.97	9d8af7f9219fe3de3d29f6b986c25a3f8d04c4672c96e4b8783023aea58c2016	26	1	Amazon	Amazon	
834	18	2025-03-03	Web Madi Toumani	-3371.00	3875769a0b0b6512e8da576eb9d1caa507d7e22cb3b2085b872ae148546eb845	31	1	Vers Livret A	Web Madi Toumani	
866	18	2025-03-13	Carte X5781 Amazon Payments Payl 12/03	-43.98	121c08be9bd591be3947480df23babcdc730cc7f21e46fcda428ea1689dc5830	26	1	Amazon	Amazon	
923	20	2025-02-05	Prlv Gc Re Keepcool Keep Core	-29.99	e55352c0479847a4c34fbfdd7978905d66186c29a98a2ca2be36fed2b69a7817	25	1	Keepcool	Gc Re Keepcool Keep Core	
910	20	2025-01-31	Ret DAB X5781 Les Ulis 2 Ga 30/01 12H36	-20.00	72c5b921b02e762fb5d1837b15dd6a6da5f758360ba44d31b2c8364583193db0	24	1	Retrait Distributeur  Les Ulis	Ret Les Ulis 2 Ga	
862	18	2025-03-11	Virement De M. Madi Toumani	150.00	9e2157e9a760e75a86a82fd4dc25c6f956ffe7f07841c71b8ce424444ebfb51c	24	1	Vers Compte courant	De M. Madi Toumani	
883	18	2025-03-20	Avoir Carte X5781 Amazon Payment 19/03	8.99	f8a681e784b47db8c2811b909f8d6a2c638f042724121a0a83cb4d9d9ff94211	30	1	Amazon	Amazon	
875	18	2025-03-17	Prlv Engie Mandat Core	-57.00	1d5f45bc5347ff2f310e9bed38f03e145cdf429c72218598c85c2dfadf8f56ef	22	1	Engie	Engie	
885	18	2025-03-21	X5781 Les Uli 20/03	-2.19	5dd341d9fe11cdf5d2f1af4f4de0c9aa5db0d002124fe01ce2a15b16a4ef12e1	24	1	Carrefour les Ulis	Les Uli	
881	18	2025-03-19	Prlv Paypal Europe $S.a-r.l. et Cie S. 1040943323391/PAYPAL567J2253W2NHA LU967ZZZ0000000000000000058 Core	-107.00	2548809c9d3fb6509004f9ea9922bbc2fc740067548da04306b0f893ad095b3c	26	1	Paypal	PayPal	
888	20	2025-01-21	X57/81 Carrefour Les Ulis 26/01	-9.54	a97106a7a179d82b728133a6af32b6730d9c56ee48894e2bbbc480f527113959	24	1	Carrefour	Carrefour	
889	20	2025-01-27	Prlv Engie Mandat Core	-8.57	75aa19861de68799f201283230a1cadc8251f55236c8cc1095fb83b94e01925b	22	1	Engie	Engie	
893	20	2025-01-29	X5781 Carrefour Les Ulis 28/01	-22.61	3fa7a592c56caa2cf0aa800abcbaf423d320645cac40a0ca751b42d6a225e436	24	1	Carrefour	Carrefour	
860	18	2025-03-10	X5781 Ubr”* Pending.uber.co 09/03	-12.00	7cfec8d0baacdcd5286473ef4dda2a0ccb2ce0165d23b50ad2e3f4cf507fa207	23	1	Uber	Uber	
868	18	2025-03-14	De Madi Toumani	60.00	716c059e0fadd05993fe0b90fafe581706d6dcb411a1e5966a82be2c7422b414	31	1	Vers Compte-courant	De Madi Toumani	
880	18	2025-03-19	Carte X5781 Sloann Paris 12 18/03	-18.00	aa5fdd1efec7d1ddd79a36c784ed84d7f4cbd4b2b2d46fa2698fa022e7446747	28	1	Sloann Paris	Sloann Paris	
884	18	2025-03-20	Avoir Carte X5781 Amazon Payment 19/03	14.99	cda5374c86e8377d1a42d622b33e50d3fb05f5945c772f24c7e806793ed9e3d2	26	1	Amazon	Amazon	
855	18	2025-03-10	Carte Amazon Payments Payl 08/03	-22.95	8da5d827894813ede6d2a73dfbb8bc4fd85ecfac3ae86368e370e28a35c652ba	26	1	Amazon	Amazon	
915	20	2025-02-03	Carte X5781 Action Les Ulis 31/01	-52.49	bcfaee88bb99b6eb9dff0fdc2d76cd63430d29cfe83134aad2928728b86952f5	24	1	Action	Action	
896	20	2025-01-29	Carte X5781 SNCF Voyageurs Paris 28/01	-22.20	429df3c3951df45f409a83d04ffda5e6da92077066237107f165a14776c4770e	23	1	SNCF	SNCF	
897	20	2025-01-29	Carte URSSAF Paca Ce Marse 28/01	-1261.00	2b74692c761d085147db631c6b21ad194312f6172174b65a1521eadb40601f01	28	1	Cotisation URSSAF	URSSAF	tax
857	18	2025-03-10	Carte X5781 Deliveroo 75009 Pari 09/03	-18.38	638c27e0ac9f296c5b88d06480a7f92fbf7853310b821c6ae03df1800dccbc61	24	1	Deliveroo	Deliveroo	
895	20	2025-01-29	Carte Nya*juste Temps Pa 28/01	-0.50	244143e6368186c029cfd6edcc2d12d7ef57d3033874e7a53540640098be9d5b	28	1	Machine à café	Nya Juste Temps Pa	
861	18	2025-03-10	Vir Inst vers Trade Republic Pea	-110.00	7f99e3971d8bf42d4a01e2a1f2c4b2e0e61c4f559bc45df96b24f15c8acb9088	29	1	Vers Trade Republic	Vers Trade Republic Pea	
903	20	2025-01-30	X5781 Service Navigo Paris 28/01	-2.50	0651a1d147a8c4534c8a759ef86d9610785c3029d58c6ca90e1702826914067f	23	1	Service Navigo Paris	Service Navigo Paris	
906	20	2025-01-30	Prlv Paypal Europe $S.a-r.l. et Cie S. 1039924947599/PAYPAL567J2253W2NHA LU967ZZZ0000000000000000058 Core	-13.49	361f2c8718d5913da3162657b7c1891d6cb7c2feb730d429f37bbfdfb2d15a74	26	1	Paypal	PayPal	
907	20	2025-01-31	Carte Amazon Payments Payl 30/01	-60.46	2957e5bb137b88dde00cc77206cb6ff0fa19c9235ad2a8dd2adfcc6aaf13ebb7	26	1	Amazon	Amazon	
908	20	2025-01-31	X57/81 Carrefour Les Ulis 30/01	-59.22	e075969a6b639c5a7d8cdbcf050128d0f25612d73b6a7be250ca921b5bf18e58	24	1	Carrefour	Carrefour	
845	18	2025-03-06	X5/81 Coinbase Ireland Dub 05/03	-50.00	f47dbc09ace206126aab806c5c67ec0d60e624be3801680e06f371014c489fb4	44	1	Coinbase	X5 Coinbase Ireland Dub	
917	20	2025-02-03	Carte X5781 Amazon Eu SARL Payli 31/01	-50.32	402a7edba6340cb78e8cf8c3465e6ccb6c05fe2330254d706be0761ce8d20277	26	1	Amazon	Amazon	
912	20	2025-02-01	Virement Mens.pel 02/25	-50.00	d37a9998b25672c90ecd7a2a06a1ecc0f5f0dd219db6e2d9d15c9f59b9492b4b	27	1	Versement PEL	PEL	savings
933	20	2025-02-11	X5/81 Coinbase Dub 10/02	-50.00	8753c170426c728e0b146fd0cd26814c7684b7928393ed30b6f4ffe9f0c717a0	44	1	Coinbase	X5 Coinbase Dub	
916	20	2025-02-03	X5781 Africa Exotique Les 01/02	-12.59	72a8ac2e7bbabee6541ce05989534069f329ff9f41f692a551ec826b9f169529	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
892	20	2025-01-28	Web Madi Toumani	-2500.00	1f6a48b9e15a9d4add99a9b9d2181ef6d3a0b8b2c51b7013751bc938aef2b69b	31	1	Vers Livret A	Web Madi Toumani	
909	20	2025-01-31	Carte X5781 Tgtg nxigw4xme10dO t 30/01	-3.99	3925ed2f02b6e30b758a6cd25ddd334d36cd4f1227b263b3795acc3e5e29f54b	24	1	T	T	
871	18	2025-03-16	De M. Madi Toumani	57.00	2173b1c191d9ec708c9866d55b067eadeb9b79fe3d2b9ce2825b658e2cbf93b2	31	1	Vers Compte courant	De M. Madi Toumani	
874	18	2025-03-17	X5781 Les Uli 16/03	-56.43	af0e3f9127222f2ba7aae0448814052ae12fdb0c9be0326fd85719e129750fbb	24	1	Carrefour les Ulis	Les Uli	
858	18	2025-03-10	Carte X5781 Sncf-voyageurs Paris 08/03	-45.00	c1a415eeac91c6f4b1bee0f2ff62f82ed119a8c14ee1d956de109b7fb035be85	23	1	SNCF	SNCF	
859	18	2025-03-10	X5781 Uber “trip Amsterdam 09/03	-1.27	162d649a28877f0e3da4b1522825d1b47ad6bc9896a2ee5ac6dd469ba1dce017	23	1	Uber	Uber	
927	20	2025-02-06	Prlv Free Telecom Free Hautdebit 135706747147939382 FR83277459654 Core	-80.60	97489e8b29ba5cdbbfedfd2c5109c38c0627cda8732d8c5a99d9c0f6baaa5a60	22	1	Free	Free	
928	20	2025-02-07	Carte X5781 Burger King Les Ulis 05/02	-13.70	fe5d085b56468258d8672634d47bedad224680688af3bff758e242d89d7928a6	24	1	Burger King	Burger King	
930	20	2025-02-09	Prlv Offre Essentiel	-5.25	2a7cce897082601a7a5a49d4675301e0b26bad417de469c0feca9c092169b0fe	30	1	Frais Bancaire	Offre Essentiel	
932	20	2025-02-10	X57/81 Carrefour Les Ulis 07/02	-53.02	7bef17282c0f1694e373c5dfaf8d296f0de25ec88eea567558872429661bd93c	24	1	Carrefour	Carrefour	
934	20	2025-02-11	Prlv Paypal Europe $S.a-r.l. et Cie S. 1040183899776/PAYPAL567J2253W2NHA LU967ZZZ0000000000000000058 Core	-31.76	b8b5a7d9660ba7478ea761483cd47a3f5da43e12022f79f08b547359d4ca59d0	26	1	Paypal	PayPal	
914	20	2025-02-03	Virement Lessonsharing MTB-2025-01-31	4070.00	ddb73132aefb650a49d4c9358299d8279feceff7fc7216490d2623cc0e061858	28	1	Lessonsharing	Lessonsharing - -	
911	20	2025-02-01	De Madi Toumani	100.00	1b2ef6b1c1dde942815376b679893919fae5871d6a59284e868459000e5d0189	31	1	Vers Compte-courant	De Madi Toumani	
543	13	2025-08-27	Vir Inst vers Trade Republic Pea	-23.00	4207b69002ff86d27ae7d1603462bb52f550902f5709d797efe256869e63c1da	30	1	Vers Trade Republic	Vers Trade Republic Pea	
716	16	2025-05-31	Virement Vir Inst vers Trade republic	-80.00	98edf50d1f889c46313d19f230987110c0bd40804b6ade132b9f8dd2ca868ee1	29	1	Vers Trade Republic	Vers Trade Republic	
422	10	2025-12-20	Virement Vir Inst vers Ben-mohamed Madi T	-52.00	7b46e46a5c37d3e0b4542480237cbcab845e5e1128e4c2103cb5f532c9e5fe73	44	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
473	11	2025-11-14	Carte X4519 Binance.com Vilnius 13/11	-650.00	286fca4a57974f387e3ea59248f1cf5aceafebab3da1078f6b0209871b080a65	44	1	Binance	Binance.com Vilnius	
938	20	2025-02-14	Carte X5781 Sncf-voyageurs Paris 13/02	-35.00	6102b31cb0421c3ce96d7381c99b11c1a79e7f1fd054272138f029cf095520ec	23	1	SNCF	SNCF	
939	20	2025-02-14	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-11.99	5fafc4ba52010881bf48cecacdc820dc575a6a6bea965b2938c8b325e9f93b2a	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
941	20	2025-02-17	Prlv Engie Mandat Core	-57.00	c04ad013d44c3c4bcdeb487b0aed3ed10ccf4422b77d5a9ef9edccc3c297228a	22	1	Engie	Engie	
511	12	2025-10-01	Virement Vir Inst vers Trade Republic Pea	-450.00	5ff2d0d43f0e13ea255b6aff83a026501e41b1a4d42ecd73ba112f821d0f4703	29	1	Vers Trade Republic	Vers Trade Republic Pea	
563	13	2025-09-07	Virement Vir Inst vers Trade Republic Pea	-5.00	2e27d0524eba819d776dbf05e719076f9ccef5b6c8b9f975e828960e3d11f445	30	1	Vers Trade Republic	Vers Trade Republic Pea	
790	17	2025-04-11	Carte X5781 Airbnb HM2YAZ4J44 10/04	-290.13	196df931011922f60895f8ee31dabdd375d9a2810012eeb1675b06168f3ff1ad	22	1	Airbnb	Airbnb	
600	14	2025-08-03	Vir Inst vers Ericia Arielle Rev	-400.00	88ff1b5d1290716a00309c74d597c9771ae8df4e55d909af176f59e6ca085f73	30	1	Vers Arielle	Vers Ericia Arielle Rev	
430	11	2025-10-22	Carte X4519 Ratp Paris 21/10	-4.50	1b029422bd62be5d0339d891ff34494caf3006a75b13f0f74b3a4d34bff42086	23	1	RATP	RATP	
624	14	2025-08-14	Vir Inst vers Trade Republic Pea	-5.00	5ad02f6cb900e4a4e16cf5c3e7904604d6d67c3e9606fa03102d4b766b41423b	44	1	Vers Trade Republic	Vers Trade Republic Pea	
626	14	2025-08-16	Vir Inst vers Trade Republic Pea	-45.00	b2bc29edbe6de91c2f850ad246297ad201865a8bc9b66c48d1f3418f11dd3627	44	1	Vers Trade Republic	Vers Trade Republic Pea	
622	14	2025-08-13	Vir Inst vers Trade Republic Pea	-22.00	a7cc4c45e14175ca3bb99d39c8a301aef711dc8cf378ee853e7eafefcd6be550	26	1	Vers Trade Republic	Vers Trade Republic Pea	
940	20	2025-02-14	Virement Vir Inst vers Trade Republic Pea	-15.00	5eb3f7d5c11e65c5f90d1e5ee46ca1b49065a995e8a46bd72ff645491a6d4fa4	44	1	Vers Trade Republic	Vers Trade Republic Pea	
523	12	2025-10-13	Vir Inst vers Trade Republic Pea	-45.00	57d98b573bfe5515e1fb2b689a517cabcc56ac7e4081a551277d4a71fdffb1c3	29	1	Vers Trade Republic	Vers Trade Republic Pea	
527	12	2025-10-14	Virement Vir Inst vers Trade Republic Pea	-100.00	b77623a400cc1c1e0aa3619749a450d75bfbe09c606cbee38b7a1a216696e350	29	1	Vers Trade Republic	Vers Trade Republic Pea	
568	13	2025-09-14	Virement Vir Inst vers Trade Republic Pea	-22.00	c18289acc08a42590a0389bc8c95c247819b6d40cccfea7db35c7637acd934c4	44	1	Vers Trade Republic	Vers Trade Republic Pea	
534	13	2025-08-23	Virement Vir Inst vers Trade Republic Pea	-47.00	4e1c67f896dad1cf0c5bfea646972e37526cc93c029e361ae438c2bfb745cc2b	29	1	Vers Trade Republic	Vers Trade Republic Pea	
590	14	2025-07-31	Vir Inst vers Trade Republic Pea	-150.00	837f4c40ca3382f57621974611cded475d2fcbe4a4d553ab488afc641dd569e1	29	1	Vers Trade Republic	Vers Trade Republic Pea	
547	13	2025-08-29	Vir Inst vers Abdou	-99.00	de96a7bdfd5441f5aa6b3ff58e35e5a15ffb87ca1e66ca75d695356f4ca65fa7	29	1	Vers Abdou	Vers Abdou	
944	20	2025-02-20	Carte X5781 Les Uli 19/02	-31.74	96d9c582e75e64b68f273eb3e72614205353c63bedf5f035fc388126b696ee96	24	1	Carrefour les Ulis	Les Uli	
557	13	2025-08-01	Virement Vir Inst vers Trade Republic Pea	-480.00	374073f60aeed6cb04b7392058886819d4d6f2fdefea5f0e824466c96d4560cd	29	1	Vers Trade Republic	Vers Trade Republic Pea	
918	20	2025-02-03	Web Madi Toumani	-3900.00	0d1e7702eb57773bfb34a45e59e8f2524155678e54548fe9ca54dd0bde2ec29b	31	1	Vers Livret A	Web Madi Toumani	
337	8	2026-07-09	Prlv Paypal (europe) $.a r.l. et Cie, 1051527750154 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-287.09	25e2859cdf4b900e84bb0698bb4f265d2425654ab84543afea3063af0b22b970	26	1	Paypal	PayPal	
902	20	2025-01-30	X5781 Service Navigo Paris 29/01	-2.50	466e8a15edc8869657f85c779d3181dd98cfbfb832b44444cca6fb49532109d4	23	1	Service Navigo Paris	Service Navigo Paris	
926	20	2025-02-06	Carte X5781 Action 4684 Les Ulis 05/02	-27.89	50a99b4e21f9d34a5f07fb6e7f2a8d223c3a8cd771501317479dee5cea1ab316	24	1	Action	Action	
922	20	2025-02-05	Carte X5781 URSSAF Paca Ce Marse 04/02	-509.00	66fb291e2c2aed611a5940040a61e81afb320e01cba27210ec20886d5120906b	28	1	Cotisation URSSAF	URSSAF	tax
406	10	2025-12-10	X4519 Paris 09/12	-0.70	6ac06a6aaba27627d4d507ac8f0c31edd712fd6fb6a5ca7162c81e38eba97d74	28	1	Machine à café	Paris	
891	20	2025-01-28	Vir Inst vers Ericia Arielle Rev	-1000.00	d6b8117c3592dedfc26061e432ed96a704aa40c5820792956cef429efb5e4c90	30	1	Vers Arielle	Vers Ericia Arielle Rev	
810	17	2025-04-18	Carte X5781 Temu.com Dublin 2 17/04	-71.80	4fea80d3c42d4e966706fa857cae33c84c2a7658291065a470242a2b9d8d92e0	26	1	Temu	Temu.com Dublin 2	
943	20	2025-02-18	Carte Temu.com Dublin 2 17/02	-19.60	d240e1a2dc9600a0e6c794cde4690384f9b72afc04959dca0f07a9ea7d5246dd	26	1	Temu	Temu.com Dublin 2	
599	14	2025-08-03	Virement Vir Inst vers Trade Republic Pea	-40.00	ea8b615dbe5725fe336bfbb0b09ba9f01cb62bdea3cddd42aab8aabff0a8cf5b	29	1	Vers Trade Republic	Vers Trade Republic Pea	
651	15	2025-06-24	Virement Vir Inst vers Trade Republic Pea	-280.00	d0037f76e9b144246b7f5f8def401176018e6f0b74b0dca986113138c5cd6f57	29	1	Vers Trade Republic	Vers Trade Republic Pea	
751	17	2025-03-26	Vir Inst vers Sakina la poste	-60.00	10b8e01933b41284ca0b8896354508510e6d9ab550eb7defd39c5ec46a609971	29	1	La Poste	La Poste	
546	13	2025-08-29	Vir Inst vers Abdou	-1.00	3883b0165b71b7aa180c51f1aac84cc085d16f0b83b0de0e8d5f91514dbf95e7	29	1	Vers Abdou	Vers Abdou	
942	20	2025-02-18	Vir Inst de Acheel France	6.97	31dbe06880167d819bc7a89fae243d90bfae172cdcd18ff850c71aad6fcd982d	22	1	Acheel France	De Acheel France	
825	18	2025-02-28	X5781 Instant Gaming Dubli 27/02	-2.29	3346372dee58029ec91f39dc5169c54c0536f9f9d9e0592ba9471af0a74bb311	26	1	Instant Gaming	Instant Gaming Dubli	
638	14	2025-08-21	Virement Vir Inst vers Trade Republic Pea	-25.00	93ffb73d7a59359e159d26dfb29cd263dd5f13106c37c9f6ae78a2de9520d47d	29	1	Vers Trade Republic	Vers Trade Republic Pea	
439	11	2025-10-27	Carte X4519 Nya*juste Temps Pa 24/10	-0.50	234c39959812e435809fc665da0c049f273831a60264447434bdb7a365c2d949	28	1	Machine à café	Nya Juste Temps Pa	
428	11	2025-10-22	Carte X4519 Nya*juste Temps Pa 21/10	-0.50	823f72381442a5e9a54fa1b0b34a834cf31e3105f742c88cc3f13c3f4a530011	28	1	Machine à café	Nya Juste Temps Pa	
44	1	2026-01-19	Vir Inst vers Madi Toumani Sakin	-20.00	2d8ee317eeeeeda4709172f5012cfe53552766d58ee9c30300c458d99b11c45d	29	1	Vers Sakina	Vers Madi Toumani Sakin	
36	1	2026-01-14	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-11.99	f0837dc01a41b58cee9af27c9e38e7a6209491b79542cad02688945a52c28274	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
132	3	2026-03-08	De M. Madi Toumani Ben-moh	180.00	a6673ae73463e756d9cbeaa7db980560196fb34751c0c55174910a0bead83fac	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
92	2	2026-02-12	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-11.99	9f0e99c8dbf5a0529630685323a5a740438ec0839ca06747be1e5b0732207532	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
161	4	2026-03-23	X4519 Service Navigo Paris 22/03	-2.55	8fa0a73a45f3a39b10cad7d715d713b7019effa309ac721bce9e71bb8cddcb4c	23	1	Service Navigo Paris	Service Navigo Paris	
5	1	2025-12-24	Virement Lessonsharing MTB-2025-12-22F:1-04-10-1	310.00	fc0c5cde9927c3ef3fdf3243c3a1ba9df208bbfe01083caa3c74ac7fe025834a	28	1	LessonSharing	Lessonsharing - - -1	
296	7	2026-06-08	Virement Vir Inst vers Ambassade de Liber	-300.00	5be9872b869c10c00e6d8ee0802d5b944ecc6cc3d49be86342af902fa2c92970	26	1	Ambassade Du Liberia	Vers Ambassade De Liber	
163	4	2026-03-24	Carte X4519 Amazon Payments Payl 23/03	-27.98	942bdb6c0e2622ab0e0f429e43c6ebb8067a8eeeba4d8e9deec6f080a1ce91dd	26	1	Amazon	Amazon	
288	7	2026-06-05	Vir Inst vers Ben-mohamed Madi T	-1100.00	cf256175c0d586ef4b4c6dba0013b5e1422acd096b2713bac6121e8f56949815	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
688	15	2025-07-17	Vir Inst vers Trade Republic Pea	-210.00	8ec77f9ddd0bc25617a15d692b966cd1e79090d2a2f21f7ce6afdbb955b11a8d	29	1	Vers Trade Republic	Vers Trade Republic Pea	
690	15	2025-07-19	Virement Vir Inst vers Trade Republic Pea	-165.00	9711b2f47dbf06840fd0dcd7a227d7b27cbb322a29e77c1006ef29d5e317e05e	29	1	Vers Trade Republic	Vers Trade Republic Pea	
921	20	2025-02-05	Carte X5781 Nya*juste Temps Pa 04/02	-0.50	19a7eb6a6bc76ecd93604629d4afe4c7fd1fcbf347f2eef2cf4db27602766bfc	28	1	Machine à café	Nya Juste Temps Pa	
936	20	2025-02-13	Carte Nya*juste Temps Pa 12/02	-0.50	390d37f300dc245669d8a7ba5bf56958db6418d086ded2a36fc85b2bf7a5e9c9	28	1	Machine à café	Nya Juste Temps Pa	
937	20	2025-02-13	Carte X5781 Nya*juste Temps Pa 12/02	-0.50	2df0efbee24addbcefd365a7b6067d89e1498d216d6d87fc1b139f63d7fdd11e	28	1	Machine à café	Nya Juste Temps Pa	
822	18	2025-02-24	Carte X5781 Ulis Viandes Les Uli 22/02	-16.00	9cc57f3d3c6de6a302127cce829f60439eece32897885f9818ebd97e48200b24	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
900	20	2025-01-30	Carte Nya*juste Temps Pa 29/01	-0.50	7bb0399a459583d436863b130d536400586c14e7867399df63f2878dd9455ff3	28	1	Machine à café	Nya Juste Temps Pa	
652	15	2025-06-24	Virement Vir Inst vers Trade Republic Pea	-15.00	9c0aac32977ce11e5ba95a5a1b0a01667c2e09411569d635779d7eaee26ed7c7	44	1	Vers Trade Republic	Vers Trade Republic Pea	
931	20	2025-02-09	Virement Vir Inst vers Trade Republic Pea	-50.00	6048b8043170bf679d5cfbb8d4e8819f25059e9ad2a993c18cee5e45f7564d99	44	1	Vers Trade Republic	Vers Trade Republic Pea	
112	2	2026-02-20	Virement Vir Inst vers Ben-mohamed Madi T	-970.00	f137e26ee024e7051eb0f734990da95ea56b06aa400f7040a16bc7d3efbb2446	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
283	7	2026-06-05	Virement Lessonsharing 2026-06-04-MTB 00006F:1-24-04-1	840.00	ef213fef5ae6785a2384d5d9790e3f5a0c4ea484fb6ab326975c1196f3594d78	28	1	LessonSharing	Lessonsharing - - - -1	
339	8	2026-07-13	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-24.14	090a53b20ac786616d0aa270fdd8e8686d782f8578ce4f21c7368e10a2fda9fc	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
679	15	2025-07-08	Pharmacie Ulis 2 Les 05/07	-46.66	1160c6f8d7463a90fa604434552464ca29d4df26a397d7bcc67363ab8ace4020	25	1	Pharmacie Les Ulis	Pharmacie Ulis 2 Les	
616	14	2025-08-09	Virement Vir Inst vers Trade Republic Pea	-35.00	ec79dbac37a06a0ca355255e8e774bf3049d177b14d0a79806a748b0b584b4af	29	1	Vers Trade Republic	Vers Trade Republic Pea	
877	18	2025-03-17	Virement Vir Inst vers Ericia Arielle Rev	-5.00	e05f942fe8e6b2ec9137e8a3a774ccbe44981c7707c3a3e64a2e57ec7110269e	30	1	Vers Arielle	Vers Ericia Arielle Rev	
446	11	2025-11-03	Virement Lessonsharing MTB-2025-10-31F:1-23-08-1	1520.00	ab0e41860e09facb628bd0fbac57f2920f0da58d5c39a019fc8ed05b2896c924	28	1	LessonSharing	Lessonsharing - - -1	
526	12	2025-10-14	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-11.99	3c1d507711c10948d2927490b42496f0cef3e9eb148f25b355783dd858f38d59	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
561	13	2025-09-03	Prlv Paypal (europe) $.a r.l. et Cie, 1044573280935 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-244.79	9e2cf7541a8d3fc221d6e2703dc6ca4187ab3ff99297a280c3d7cff49c406cd8	26	1	Paypal	PayPal	
608	14	2025-08-05	Prlv Paypal (europe) $.a r.l. et Cie, 1043948814696 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-244.79	39971deff11b6a8e19bbd3d778e9165bda72c767dbe39e069b98b7e29acba5c4	26	1	Paypal	PayPal	
617	14	2025-08-11	Virement De M. Madi Toumani Ben-moh	250.00	fb0a1a98daa380e030c5eb77e1365e095782e6a91850147dadad34db32cfcefd	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
541	13	2025-08-26	X7303 Sc-elyians Les Ulis 25/08	-5.93	21934ede79ebee7c6059ddee8777e1dcb8fdfb895d857959643fde4025a142b4	24	1	Les Ulis Sc-Elyans	Sc-elyians Les Ulis	
672	15	2025-07-05	De M. Madi Toumani Ben-moh	500.00	f24e917904bd7221318e499f08236da22c6531c37ac3e574a7698f447890366b	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
722	16	2025-06-02	Vir Inst vers Madi Toumani Sakin	-40.00	03e631ef240726f7efbef21cae3066bf8a7560433a5c23d77e2bde8ed7ac28f0	29	1	Vers Sakina	Vers Madi Toumani Sakin	
723	16	2025-06-02	Virement Web M. Madi Toumani Ben-moha	-50.00	b3f7e097dca2a905a597580b8e9daae4664ea6df810a675c7e1c0cee8cdb8d76	31	1	Vers Livret A	Web M. Madi Toumani Ben-moha	
763	17	2025-03-31	Carte X5781 Ulis Viandes Les Uli 29/03	-23.20	4dc7abf140f7ed6a122e1bcbce8ba9061d37e048f378ac430e719c8e142053ca	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
894	20	2025-01-29	Carte X5781 Nya*juste Temps Pa 28/01	-0.50	a2339835c37eed38a3f97afc214cfd13c374f20346d7ce88ab89a6d146631ac7	28	1	Machine à café	Nya Juste Temps Pa	
864	18	2025-03-11	Prlv Paypal Europe $S.a-r.l. et Cie S. 1040738636575/PAYPAL567J2253W2NHA LU967ZZZ0000000000000000058 Core	-60.00	19744ad0caf8b77ffa3db069cd8274d1ca9cb8eb4d63ffb79497652dcb05b1cb	26	1	Paypal	PayPal	
899	20	2025-01-29	Vir Inst vers Madi Toumani Sakin	-50.00	a0300c32758fb4e7920cec63391450d38cddb59a94e296c9b7d34fc5be163443	29	1	Vers Sakina	Vers Madi Toumani Sakin	
924	20	2025-02-05	Prlv Paypal (europe) $.a r.l. et Cie, 1040058197822 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-221.02	74a398b9a9ff7cc221206726ae5110918e3bac58a65ad8a01546467b5fd63e42	26	1	Paypal	PayPal	
479	11	2025-11-21	121.11 Vir Inst vers Sakina Madi Touman	-150.00	9635d7cc49f1141c717a8ec52e7f6ad55253b75f73a72f859d38faa9969abb53	29	1	Vers Sakina	. Vers Sakina Madi Touman	
695	16	2025-05-22	Virement Vir Inst vers Trade Republic Pea	-705.00	ae3a38a769d88cf43a3254de8787b269fe40eb6c213771980580fec36446e4ba	29	1	Vers Trade Republic	Vers Trade Republic Pea	
870	18	2025-03-14	Virement Vir Inst vers Trade Republic Pea	-102.00	65ace9ed408b6b741624ec6411f8d1c86b95834b145d6fc7040fc697007816f0	29	1	Vers Trade Republic	Vers Trade Republic Pea	
655	15	2025-06-27	Virement Vir Inst vers Trade Republic Pea	-100.00	ad286908c54b967b4ede58f99681c67865e95fdf4944efc2171a354d870607bb	29	1	Vers Trade Republic	Vers Trade Republic Pea	
665	15	2025-07-02	Virement Vir Inst vers Trade Republic Pea	-50.00	b6a202adb5759a858625713dd796f67527fdb564a8fa4032242bbe8cd5c1fcec	29	1	Vers Trade Republic	Vers Trade Republic Pea	
673	15	2025-07-05	Virement Vir Inst vers Trade Republic Pea	-250.00	dab984db76bbd88839c44a51efbf5f2e639cfcf66793c0750a6ba9102dde68a4	29	1	Vers Trade Republic	Vers Trade Republic Pea	
677	15	2025-07-07	Virement Vir Inst vers Trade Republic Pea	-1.00	cffe4fce824038c9512cb31f0e673d48d5aa328b28284990f71c839dbca18ea1	29	1	Vers Trade Republic	Vers Trade Republic Pea	
799	17	2025-04-14	Carte X5781 Prestige Kebab 69007 13/04	-16.00	77f4ea4f9a3de246de882164605cb41b46fa069cc174571eb8ec3fe60937899c	24	1	Prestige Kebab	Prestige Kebab	
207	4	2026-04-21	Prlv Paypal (europe) $.a r.l. et Cie, 1049729158879 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-162.49	159d8fe2589ae086751de28e0fb6f04a7ff4d136d8e763f522631ac3349c1f2e	30	1	Paypal	PayPal	
514	12	2025-10-03	Prlv Paypal (europe) $.a r.l. et Cie, 1045231950478 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-244.79	839cbe77440978a30090c0709272f6083b4fb9f2c05fcc182f70fde6f52c7f97	26	1	Paypal	PayPal	
783	17	2025-04-08	Prlv Acheel France REF-4A34CO0OED35EA6GBEDFAGFGWBH4 Core	-7.02	89d90be801396bd6f66437984e9b6988809b44fc750dd684b962843a64b16bb1	22	1	Assurance habitation Acheel France	Acheel France Core	
659	15	2025-06-30	X/303 Boucherie Marche Les 28/06	-24.70	6311a80c0acfbfb128c34bf3ad1c89356fc255a7a85fae60620c1cce79dc1045	24	1	Boucherie Marche	X Boucherie Marche Les	
24	1	2026-01-08	X4519 Nyx*stcommerce Paris 08/01	-0.70	8a8d7d8b8dac0b0dc67bc4bcb9b2300ccad4d7827e62f62ce8233e20ac98a197	28	1	Machine à café Paris	Nyx Stcommerce Paris	
20	1	2026-01-05	Vir Inst Wero vers Vadijiguiba Di	-10.00	0b700f76c19e6b1ad7f0a3d0b65b9639ab173ba137baae8147fada76804c1a53	23	1	Wero Vadijiguiba	Wero Vers Vadijiguiba Di	
126	3	2026-03-04	Web M. Madi Toumani Ben-moha	-1000.00	1f66c389b93a6bce081a6cc1051d55f8b214557d27b4f04d2fe601e4919f927a	31	1	Vers Livret A	Web M. Madi Toumani Ben-moha	
127	3	2026-03-05	Prlv Free Telecom Free Hautdebit 145690719847939382 FR83277459654 Core	-29.99	53d3100e5159c67f414113ed79aa2cc98469dac5bc190a93ea1529238fd3ea45	22	1	Free	Free	
177	4	2026-04-02	Carte X4519 La Toscana Paris 01/04	-15.00	8b6e15785e236302e1c7663cb7dda348891082d32c575cd788247b8dfcb0f61e	28	1	La Toscana Paris	La Toscana Paris	
390	10	2025-12-02	De M. Madi Toumani Ben-moh	600.00	c105c21cb3e9bb0b79d25d310dcf0739f5145afa39a42c7d7030e8b8ba394ed2	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
402	10	2025-12-09	Prlv Acheel France REF-BOOEDO69D13C448078GFGWBH4 Core	-11.44	ca14ca7023172420ff513b514f2f5d7480dd304904a7f2e3902ef1a6999f1c9a	22	1	Assurance habitation Acheel France	Acheel France Core	
440	11	2025-10-27	Carte X4519 Uber “trip Amsterdam 24/10	-1.65	26b9505a6c1666ba6708c0cecb82925f5676ca5c8b31d67bd2a1ad95fe6544c0	23	1	Uber	Uber	
17	1	2026-01-05	Carte X4519 Nyx*stcommerce Paris 05/01	-0.70	7f1713f944d57c5545f1448ddc77137fc36ad1635921266f722c9e251eb3af9f	28	1	Machine à café Paris	Nyx Stcommerce Paris	
220	5	2026-05-05	Prlv Acheel France REF-81821F2DAF9EDES013GFGWBH4 Core	-12.47	8461738f9a05aca0ec6c046b170eafff5a9463e6cb65ba36f752d6424b308d21	22	1	Assurance habitation Acheel France	Acheel France Core	
398	10	2025-12-08	Vir Inst vers Ben-mohamed Madi T	-15.00	9639aff2ecd2626bd653d137a7293354c1e1a6ed3cbf819b0c3b0f026bdac93c	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
725	16	2025-06-03	Virement Vir Inst vers Trade Republic Pea	-330.00	7dfab7ef6acefd282b3a9c367751cbcfc352ae9f3a5557640464d390cb2313d9	29	1	Vers Trade Republic	Vers Trade Republic Pea	
731	16	2025-06-09	Virement Vir Inst vers Trade Republic Pea	-385.00	94fe1db6301dfb3a94112bb72f7237eab6e39113be674857f015b89507fb40a1	29	1	Vers Trade Republic	Vers Trade Republic Pea	
746	17	2025-03-22	Virement Vir Inst vers Trade Republic Pea	-30.00	e54a29d7c94e8cfcba80ee79bffe75a060603fb575d763f45ea95e49f5ab6c22	29	1	Vers Trade Republic	Vers Trade Republic Pea	
754	17	2025-03-27	Virement Vir Inst vers Trade Republic Pea	-30.00	0f681c3be10acd00d9eeaeed74a8353023da0a1e9d1bc167c5cfe7532729f31d	29	1	Vers Trade Republic	Vers Trade Republic Pea	
795	17	2025-04-12	Vir Inst vers Trade Republic Pea	-20.00	37b6cfcf2acf1e0a6fdbf1bde655e2960fff04caa1f0061007ceacd0e206f08a	29	1	Vers Trade Republic	Vers Trade Republic Pea	
804	17	2025-04-15	Virement Vir Inst vers Trade Republic Pea	-10.00	6c98849a2b75b2e4103d11e020dda1787b239e0e3c150c5d9ce4edf0b88a3318	29	1	Vers Trade Republic	Vers Trade Republic Pea	
811	17	2025-04-19	Virement Vir Inst vers Trade Republic Pea	-10.00	1408547f1fd88087e1104e08d10950ac99474f2fa23d5fd347b9627ca4e7b950	29	1	Vers Trade Republic	Vers Trade Republic Pea	
813	17	2025-04-21	Virement Vir Inst vers Trade Republic Pea	-350.00	543a276ae0af5f38bc8dad97234ae8cccc7156e82063753f8e62b6958111eebc	29	1	Vers Trade Republic	Vers Trade Republic Pea	
876	18	2025-03-17	Vir Inst vers Trade Republic Pea	-30.00	867f704703037002a56ac00b7149640a1c05d70cca4098cd3abd53d93df5e2cf	29	1	Vers Trade Republic	Vers Trade Republic Pea	
878	18	2025-03-17	Vir Inst vers Trade Republic Pea	-216.00	dc296ca34568f9990357ed5a3fbf54315f3bf1a936e914e637dc2845c77e0e1a	29	1	Vers Trade Republic	Vers Trade Republic Pea	
882	18	2025-03-19	Vir Inst vers Trade Republic Pea	-50.00	c0bd898f8048908ff2862192fd7a4871768b7da37a2246ff99ea89bc05438902	29	1	Vers Trade Republic	Vers Trade Republic Pea	
683	15	2025-07-11	Virement Vir Inst vers Trade Republic Pea	-130.00	375bca51af1e0a9718c0b36637f746db9e398cdac78a59d412589d9c2dc70f37	29	1	Vers Trade Republic	Vers Trade Republic Pea	
753	17	2025-03-21	Vir Inst vers Ericia Arielle Rev	-600.00	849324e224fe700666277188cc0a1539d03e64c160bc38d9655917d252e6f504	30	1	Vers Arielle	Vers Ericia Arielle Rev	
828	18	2025-02-28	Vir Inst vers Ericia Arielle Rev	-600.00	557e286d2aa51c05fe2d63c098eb2de4df15834bf427e9ad0f47724bd0d05292	30	1	Vers Arielle	Vers Ericia Arielle Rev	
925	20	2025-02-06	De Madi Toumani	120.00	f94cbc1f3324b4b4e1deabb23f5aff93d878cc07169548d67bfd6b57b1284233	31	1	Vers Compte-courant	De Madi Toumani	
850	18	2025-03-07	Virement De Madi Toumani	300.00	d444cea660775e23a1125663dc512434a0ffbb65ee535a19ca06ebe3d1fcca38	31	1	Vers Compte-courant	De Madi Toumani	
405	10	2025-12-10	Carte X4519 MAG0118 Les Ulis 09/12	-125.97	c85f52d2e3d2ecb3ca7b2bce0d38c284bd57b3faf2afb19398346939a86782ea	24	1	Carrefour Les Ulis	Les Ulis	
714	16	2025-05-31	Virement Vir Inst vers Trade Republic Pea	-79.00	e8e93cd50b2500ea39796c7c5aeb2be7ab273eba7425b70e057b4d3617c82bb0	44	1	Vers Trade Republic	Vers Trade Republic Pea	
792	17	2025-04-11	Vir Inst vers Trade Republic Pea	-15.00	15325091d34040b6a0db6d5b22aca2717bce8e8d78ae6bc9882970da72a223f4	44	1	Vers Trade Republic	Vers Trade Republic Pea	
788	17	2025-04-10	Virement Vir Inst vers Trade Republic Pea	-20.00	a9ce21cbf6d9b18d2359683b1b2c9cbc752551382910d0383e5995a74fd3bcc7	44	1	Vers Trade Republic	Vers Trade Republic Pea	
768	17	2025-04-01	Virement Vir Inst vers Trade Republic Pea	-50.00	efd44e35dab558d6be33fd662c3bcecb4ff97513f941c7afbd2c6a6c7659ccd9	44	1	Vers Trade Republic	Vers Trade Republic Pea	
415	10	2025-12-15	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-11.99	af7c12e6e5ccac36b74b6ef1d001d96d6b8eca1413b13ff0deb296b3fae7a074	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
475	11	2025-11-15	Vir Inst vers Ben-mohamed Madi T	-30.00	5ab0ac26679642b2aec7f69d5681762c9bea2cb08e8bdbd624e4fb93ef90a7c7	44	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
465	11	2025-11-07	Prlv Acheel France REF-1E86B3646431168C51GFGWBH4 Core	-11.44	26f9316e43cd6fae96096dbc7641aeba5d809ac7da7ba3429eaa2d062acaeae6	22	1	Assurance habitation Acheel France	Acheel France Core	
507	12	2025-10-01	Virement Lessonsharing MTB-2025-09-30F:1-01-07-1	840.00	d7153884c057edfac65f394ff3edcfaeaa7a84791969af1215e5d8c158914c89	28	1	LessonSharing	Lessonsharing - - -1	
517	12	2025-10-07	Prlv Acheel France REF-E4975698291BFO3A7AGFGWBH4 Core	-11.44	e40d5e04d069f377cc250bde3961df8213a43967a3cea5ecf0738ff795218ff8	22	1	Assurance habitation Acheel France	Acheel France Core	
559	13	2025-09-02	Vir Inst vers Madi Toumani Sakin	-120.00	0b94a5f1f836084022b8dac535120b738eee5b05b23fb3ec2e3e36ee67016cf2	29	1	Vers Sakina	Vers Madi Toumani Sakin	
610	14	2025-08-06	Prlv Free Telecom Free Hautdebit 140382840247939382 FR83277459654 Core	-19.99	67a38193269780e3205b56e8487ea1daabcafba3e5ca5d229e7a1ac70b6d5d6e	22	1	Free	Free	
613	14	2025-08-08	Vir Inst vers M.madi Tarik	-280.00	c32ea4c1e31cde10afa39cd34f8c74d4c576cdcf59371d9bf0e3e75187962cc5	29	1	Vers Tarik	Vers M.madi Tarik	
7	1	2025-12-29	Carte X4519 Uber “trip Amsterdam 28/12	-5.19	04000d099619dad07b35ac6ea1c0d6faaede2ea8d0d406e8bc4f23a153bdaaff	23	1	Uber	Uber	
35	1	2026-01-14	Carte X4519 Nyx*justeatemps Rois 14/01	-0.50	ab519dc46e22d2f89b0fd85827ee648e062371229c10f381daee8e8082b87929	28	1	Machine à café	Nyx Justeatemps Rois	
95	2	2026-02-15	Vir Inst vers Ben-mohamed Madi T	-42.00	fe231ea8592f2bd97aeb28469fe4fb52338ff8eb9ea9f7f8fe9b2349ad0816fe	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
727	16	2025-06-06	Prlv Acheel France REF-5C3D243DD54D01ECB8GFGWBH4 Core	-11.44	7c78c0f6364812afaa7bddb3af92be2d2bea546072be368a35c5c6b8f2248f4d	22	1	Assurance habitation Acheel France	Acheel France Core	
740	16	2025-06-16	Carte X7303 Ulis Kitchen Les Uli 13/06	-12.60	3bd3ef2db1f70935c800389aa75ceb578d3d7866e7879517eac0413a1805f707	24	1	Kitchen Les Ulis	Ulis Kitchen Les Uli	
766	17	2025-04-01	Prlv Paypal Europe $S.a-r.l. et Cie S. 1041193368510/PAYPAL567J2253W2NHA LU967ZZZ0000000000000000058 Core	-13.49	7e75d953a2fac716720724d84297245841ce17ed58991f7aa715d602bd8a4548	26	1	Paypal	PayPal	
779	17	2025-04-06	Virement Vir Inst vers Trade Republic Pea	-15.00	1f4cdf7b44e00f161dbf0d297c4bc3a3206ed0887a6b82421b9cf3792bf0adde	29	1	Vers Trade Republic	Vers Trade Republic Pea	
831	18	2025-03-03	Lessonsharing Salaire Février 2025	5060.00	427742ecd5f421057e2a7ea3012359f50df567ff8b7b9c33e2a575194cdf3bc1	28	1	Lessonsharing Salaire Février	Lessonsharing Salaire Février	
865	18	2025-03-12	Carte Amazon Payments Payl 11/03	-14.99	12e1c6bbeeef9adf51afb54a4a4cc2446a845bde6bb9fb970d9d7ed9f710e217	26	1	Amazon	Amazon	
890	20	2025-01-28	Virement France Travail 2502450012634 3U 8736222U 28012025	5255.87	c6b14e8c2feac8977f0b63c3986946fadc40d65576ab8d22e074975ace0e83a1	28	1	Allocation France Travail	France Travail	benefit
898	20	2025-01-29	X5781 Wakanda Factory Pari 28/01	-8.90	637634dedd451db210cf23c01c9ffb9c39235c5091fb3b4459e3030d63c16bc9	24	1	Wakanda Factory Paris	Wakanda Factory Pari	
644	15	2025-06-21	Virement Vir Inst vers Ericia Arielle Rev	-100.00	a9794eba5cd6c68971a16be914df756e77a3e51a5d7f3dc5193f66219f4817bf	30	1	Vers Arielle	Vers Ericia Arielle Rev	
905	20	2025-01-30	Carte Trade Republic Berli 29/01	-50.50	bcbe0ec6161bdbb20f195c7117a439c812983866e7e4838cdb2d18081ed92de3	44	1	Vers Trade Republic	Trade Republic Berli	
715	16	2025-05-31	Vir Inst vers Trade Republic Pea	-1.00	56ca2b5d2265266281bd04dea7e042dc648b99f31408d59ee2c909a0a0f1cf98	29	1	Vers Trade Republic	Vers Trade Republic Pea	
678	15	2025-07-07	Virement Vir Inst vers Trade Republic Pea	-250.00	9efd4a12ea81504224709e0cec207ee8e140c96c6188908ac5e6a0fa4c614564	26	1	Vers Trade Republic	Vers Trade Republic Pea	
800	17	2025-04-14	Carte X5781 Smile World Malakoff 12/04	-10.00	1c654c72aad7236d9fd7ad5df59b2554b2a146a107afd03569b71f4962f2e030	33	1	Smile World	Smile World Malakoff	
913	20	2025-02-03	De Madi Toumani	500.00	b134ae9ad5e745cc1ca30577a6f0a8c1b4ab8271c60ce9cecb3464f8a30dd567	31	1	Vers Compte-courant	De Madi Toumani	
929	20	2025-02-08	Virement De Madi Toumani	150.00	746443eaaffc033e84a599133a98088aa25361aa7d1343f3c2d1f7b47dc229f9	31	1	Vers Compte-courant	De Madi Toumani	
935	20	2025-02-13	De Madi Toumani	350.00	bcffbd068ed457eb614372f31b9503cfac0f31fc4ca71a3e17e6684f161c62fe	31	1	Vers Compte-courant	De Madi Toumani	
824	18	2025-02-28	De Madi Toumani	600.00	d94e779edfa37c18817423555dfdf5ab19cb8197db27d10a0400ff51de24e0a3	31	1	Vers Compte-courant	De Madi Toumani	
872	18	2025-03-17	Virement De M. Madi Toumani	200.00	8088cd39ad764f1e75f066ba063c277440a6333821dfce6fa66deb4442cd8607	31	1	Vers Compte courant	De M. Madi Toumani	
873	18	2025-03-17	Virement De M. Madi Toumani	220.00	70da48d90efa0d3180809efefd43f9740acd92e07d64c31131d5b930b52b852b	31	1	Vers Compte courant	De M. Madi Toumani	
879	18	2025-03-19	De M. Madi Toumani	60.00	c9940ffcf4ac25744d82994379d98f6b94e90f6bcad1332540313ac43d32597d	31	1	Vers Compte courant	De M. Madi Toumani	
867	18	2025-03-14	De M. Madi Toumani	100.00	000f7ee5dd8878eb5966813ceb6a3f98bbaa795f8a2723ed0bf5bfbfdb30705d	24	1	Vers Compte courant	De M. Madi Toumani	
476	11	2025-11-15	1 15.11 Web M. Madi Toumani Ben-moha	-2100.00	5af89c6fb369e0b08598363eca31d2e853a29c930c86120c504fd4f267d29458	31	1	Vers Livret A	1 . Web M. Madi Toumani Ben-moha	
487	12	2025-09-24	Carte X7303 Paris 23/09	-0.70	3d18b9eaf7f4e8d8ac3c041f25042097c5220326aded4c4dd4947cfce935be49	28	1	Machine à café	Paris	
316	8	2026-06-25	X4519 Arcueil 24/06	-3.04	ed54c3933787fcd779fefe2d1e42147ee9828ca1f502ed1d4606dd39cd6b712e	28	1	Gare Arcueil	Arcueil	
21	1	2026-01-06	Vir Inst de Mile Aya N’goran	900.00	2b6f85f955b364cc1e2b36f9a8bc4f2c8f7ee133cd93e250ccfe74d8f9e08185	33	1	De Sonia	De Mile Aya N’goran	
661	15	2025-06-30	X7303 Service Navigo Paris 29/06	-22.20	57c335a837e992ad72cfa5d1ddb6ee1306a071c27f59d350a3aaf18a714bd285	23	1	Service Navigo Paris	Service Navigo Paris	
103	2	2026-02-17	Carte X4519 Ulis Viandes Les Uli 16/02	-37.38	103ac2eddd2cebea608f81032068404af04369678b899f5e30ac386e78cf9705	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
70	2	2026-02-03	Virement Lessonsharing MTB-2026-01-30F:1-08-11-1	1050.00	b51e4c4ddc7e13fe3a08351d158966c9a06a9e54e5256eff2b31d08149a34113	28	1	LessonSharing	Lessonsharing - - -1	
77	2	2026-02-03	Virement Lessonsharing MTB-2026-01-30F:1-22-11-2	590.00	5091b843f3cca5cc5f11363889b35bb22a0a1a9cab6163d044031b3b82871c44	28	1	Lessonsharing	Lessonsharing - - -2	
128	3	2026-03-06	X4519 Carrefour.fr Mondevi 05/03	-63.18	cfb639b497c3e7e6bc7e613ae02e21efc451996844d125412549d7dbd9ef7a9f	24	1	Carrefour	Carrefour	
219	5	2026-05-04	Carte X4519 Ulis Viandes Les Uli 02/05	-35.73	1f42410c3c30c5b42bcdb6bdb56181d788a3abb811c33a1865a18afc08e905ad	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
293	7	2026-06-08	Carte X4519 URSSAF Idf Montreuil 05/06	-358.00	5689924b29524d95ab74ba17d342ffe75bc5ad6c8d28c6240f84f0fe648538c6	28	1	Cotisation URSSAF	URSSAF	tax
41	1	2026-01-17	Virement Vir Inst vers Ben-mohamed Madi T	-500.00	4be15193150936f1ea129e20de9fde2a44e18ea37762450d84036ec4894d6d49	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
114	3	2026-02-23	Virement Vir Inst vers Ben-mohamed Madi T	-20.00	ef1225328026d19f6fd4f5b5b2bb6825206bb45022b35e47952ea7bf8cbecb74	44	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
197	4	2026-04-13	Prlv Paypal (europe) $.a r.l. et Cie, 1049504421276 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-195.95	a05c794202f2880e4004b4ae7a7dfbec12e36a0b6c0208bc99e7edbb0b180194	29	1	Paypal	PayPal	
657	15	2025-06-28	De M. Madi Toumani Ben-moh	30.00	93d0b82d91c465181dc1853d2dd09effd52b24947284b12bc01b9da4201c32f4	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
711	16	2025-05-30	Virement Lessonsharing MTB-2025-05-30F:1-19-04-1	1040.00	e244ac95c2c64b44197290f42c3afdb04c2e17df9dd571273c105ef9cd3c2d20	28	1	LessonSharing	Lessonsharing - - -1	
474	11	2025-11-14	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-11.99	1cfa9947f49c17be32fed4ad251ac0f89cbf43f9164cbec2cda95c4cd8567e87	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
481	12	2025-09-22	Virement Vir Inst vers Trade Republic Pea	-10.00	a06416c59d22c853e3f7c0919321b4aa3aa8e4bd44b8ef97bc23ab3f3e98ffc4	29	1	Vers Trade Republic	Vers Trade Republic Pea	
536	13	2025-08-24	Vir Inst vers Madi Toumani Sakin	-50.00	f412c92bf8361504a01e33cc1cffd14bb2af0cdbd0e919efeb27c2c2e93cbb56	29	1	Vers Sakina	Vers Madi Toumani Sakin	
553	13	2025-09-01	X/303 Phcie Courdimanc Les 30/08	-17.90	00444d8256c54905e208a4a958bf9da849be708b546c1b3c221b6049fca3378b	25	1	Pharmacie Courdimanc	X Phcie Courdimanc Les	
586	14	2025-07-31	Virement Lessonsharing MTB-2025-31-07F:1-01-05-1	620.00	1dc0c7d8e3c231603bff2fbadfa70291ca6c0d0854f88fe7437cf77f0c5f5512	28	1	LessonSharing	Lessonsharing - - -1	
631	14	2025-08-19	Prlv Paypal (europe) $.a r.l. et Cie, 1044210876283 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-222.10	fa4a350aad039474eb1ea6c7577cd693f8c54bbf3979c93935d052f273b812e2	26	1	Paypal	PayPal	
641	15	2025-06-21	De M. Madi Toumani Ben-moh	89.00	3b76a366e600a7e4ab5df9a8cedd16884238ea5d1958eff6380e1b51f60efcac	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
666	15	2025-07-03	De M. Madi Toumani Ben-moh	2850.00	049f231cc578ceb3930230b7db583152b18b75fb3ce4252a50dce303a944e1ad	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
728	16	2025-06-06	Virement Vir Inst vers assjad	-8.00	b56a012078c7554d72db515af14dddfb340c7bf760c8a0263619bbb4999dfb03	29	1	Vers Assjad	Vers Assjad	
667	15	2025-07-03	Prlv Intérets débiteurs Période 2 Trim 25 TAEG=20,02%	-0.02	dcb9202f84e57ac9939d458b76c5b4130c21fb03cde0f835550ce8b455508228	30	1	Intérets Débiteurs	Intérets Débiteurs Période 2 Trim Taeg= , %	
748	17	2025-03-25	Prlv Paypal Europe $S.a-r.l. et Cie S. 1041071115045/PAYPAL567J2253W2NHA LU967ZZZ0000000000000000058 Core	-7.18	2f579637c120e33c1eecea232d14e77b643d207799119ac8b22c929d7d930b10	26	1	Paypal	PayPal	
798	17	2025-04-14	Carte X5781 O’delices 56 Malakof 12/04	-19.00	5df2f1fe5955000211dd553e5fbfa52f48deb081573720146d91317e0f2fc7e1	24	1	O’delices Malakof	O’delices Malakof	
805	17	2025-04-17	Virement De M. Madi Toumani Ben-moh	100.00	6bb435bbf1d9948970c6b9b1ccc16ea74a466c237ed6ea819f13b795cffdc7fb	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
849	18	2025-03-06	Prlv Paypal (europe) $.a r.l. et Cie, 1040680208419 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-221.03	d21ea74c60110dbf5ecf51dd48d51d2cc3276f431c53cf7f32a6cd88ecedac0f	26	1	Paypal	PayPal	
833	18	2025-03-03	Vir Inst vers Trade Republic Pea	-50.00	df4b70ba675797f7fa93f27a7183171552c4b96fba863c1fbd14019c453338ea	29	1	Vers Trade Republic	Vers Trade Republic Pea	
919	20	2025-02-04	Carte X5781 Nya*juste A Temps Pa 03/02	-0.50	b9c88ac37597398cdd3908469b83848e91b3f2b188cca699dbc38aec036f97de	28	1	Machine à café	Nya Juste A Temps Pa	
589	14	2025-07-31	Virement Vir Inst vers Trade Republic Pea	-445.00	47eb7c055c235ee169f7d37b35c1c960b0c7fce86a853c0c54a832f5ac683399	29	1	Vers Trade Republic	Vers Trade Republic Pea	
598	14	2025-08-02	Virement Vir Inst vers Trade Republic Pea	-60.00	90b9926756307ae8b7f7ad62b06aba767abe86dc505beec354a963c81532f0d0	29	1	Vers Trade Republic	Vers Trade Republic Pea	
344	8	2026-07-17	Prlv Idfm IDFM/002677230-5626153-2678663/MAND ATE-REFERENCE000000000000556877 Core	-17.96	069cdc9eb85f10e6b6dcc5310ae394b9f28f17c60f63d954deec281b3f038cc3	23	1	IDFM Core	Idfm Idfm - - Mand Ate- Core	
272	7	2026-05-22	Prlv Idfm IDFM/002073317-4326232-2073281/MAND MANDATE-REFERENCE00000000000055687/7FR42277457385 Core	-20.00	b1bf035f7ccf25d81f4ce7e2bbb7725c64ef48d528daf97ad5da4ef58a527e9c	23	1	IDFM Core	Idfm Idfm - - Mand Mandate- Core	
40	1	2026-01-17	Virement Vir Inst vers Ben-mohamed Madi T	-80.00	8288e9a089a257b56a8401597bd55895dc263637a82bd23515acafdfea2531b3	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
585	14	2025-07-31	Virement France Travail 2521050017034 3U 8736222U 31072025	5255.87	c78d511db5a2d87123bab50925105e1aa4dab214fdd858d44ce965d4f6efa59b	28	1	Allocation France Travail	France Travail	benefit
945	21	2026-07-22	Carte X4519 Cofidis Amazon Ville 21/07	-63.68	785165aa819f9c68ca394a30894ae4c4fced512c9503210ff7d730db58490050	\N	1	Amazon	Amazon	
381	10	2025-11-24	Carte X4519 Burger King Les Ulis 21/11	-12.60	c2a00a8bef221767200ad87f47ade3512f7e730d42b2cd3699fdbe18aff7b25c	24	1	Burger King	Burger King	
869	18	2025-03-14	Prlv Bouygues Telecom 602100 06xxxxx951BT1150PNIAM30 Core	-11.99	e9e1b7e586dc4be12b34091a6f952db24386ab66dd16e0ee7ac335f66a909907	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
278	7	2026-06-04	Prlv Acheel France REF-996928EF59FA47A80BGFGWBH4 FR4527787F911 Core	-12.47	3ba6a9b63eb92b5bc1d8d0832321a5f701a9723785544c19327c0d564ceb4dce	22	1	Assurance habitation Acheel France	Acheel France Core	
295	7	2026-06-08	Prlv Paypal (europe) $.a r.l. et Cie, 1050779745906 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-391.89	1f0b39e26d4bc939f8a273d0a9435f26a7da419e1f1815a90c99544baeea5c0d	26	1	Paypal	PayPal	
15	1	2026-01-02	Virement Vir Inst vers Ben-mohamed Madi T	-210.00	26fac8ebc18444bb2843cf37e1c6158c8d6f07b57c0f93812e754c176bbc64c1	26	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
399	10	2025-12-09	De M. Madi Toumani Ben-moh	100.00	0a2a852d53904004f20f5b92f4a7e5bfff98835ffe1e5c2405063ed24c39e3d0	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
432	11	2025-10-22	Vir Inst vers Ben-mohamed Madi T	-5.00	32877fb0be0a963f8c304c8e2417a5950cca3bcee2a998593c388b6c8d19d436	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
947	21	2026-07-27	Carte X4519 Africa Exotique Les 24/07	-6.13	2357aac887e191d3fde4ec876bb8ce5d39dff21448974e9fae32a6c256c9bd2a	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
946	21	2026-07-24	Virement De M. Madi Toumani Ben-moh	100.00	304e2d350e2d4775a6e9cb5f5c16c2d1bcf46a7693ff000d38b8e9298f04f9ff	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
281	7	2026-06-05	Virement Lessonsharing 2026-06-04-MTB 00005F:1-17-04-1	1520.00	762c5be0493b8f4fb4c0d5a21ef95beb7425735eecdd9dd1231eb85239f887be	28	1	LessonSharing	Lessonsharing - - - -1	
948	21	2026-07-27	Carte X4519 Ulis Viandes Les Uli 24/07	-15.26	53185e5ac060940f379aecc14666da4e11068c0c89107c78750e64c13762b27f	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
974	21	2026-08-11	Carte X4519 Labo Delaitre Les UI 10/08	-24.96	1d234b4ec8b60aeac386d0cc0cf0372876d424e31d9443b69ae128dc71ba1461	25	1	Labo Delaitre Les Ui	Labo Delaitre Les Ui	
960	21	2026-08-04	X4519 Amazon Prime Fr Payl 03/08	-34.95	435968a05ebcf070974ae33e4d3056e01a22eb44f42c32beaa33f4cc66a5488a	26	1	Amazon Prime	Amazon Prime	
956	21	2026-08-03	Vir Inst Wero de Fazal El Karim	1.00	62195ee87e7f789538772c6a146d27574402d3c19bc8bd8abd6a0d30319bd0e8	29	1	Wero De Fazal El Karim	Wero De Fazal El Karim	
949	21	2026-07-30	Carte X4519 Nyx*justeatemps Rois 30/07	-0.50	a70f9d2e15ebcdb831c0c9c4f16065b69fbbe052e68a008c831cb5a5f3452680	28	1	Machine à café	Nyx Justeatemps Rois	
950	21	2026-07-31	Carte X4519 Nyx*justeatemps Rois 31/07	-0.50	d6e0b04e5118efd65f1019965e459a97fce24eed5ad4aa0b8e002c41b721455c	28	1	Machine à café	Nyx Justeatemps Rois	
951	21	2026-07-31	Carte X4519 Nyx*justeatemps Rois 30/07	-0.50	65f378ce2a565178c669420ae4090a287ddcc5b7c362d258426ce077a5880c28	28	1	Machine à café	Nyx Justeatemps Rois	
952	21	2026-08-01	Mens.pel 44690429796 08/26	-50.00	fa08d230f89da4a31ac3ff779ae6bc0180df72e0bb8fd587b95fa1674da12a09	27	1	Versement PEL	PEL	savings
953	21	2026-08-03	Virement Lessonsharing 2026-07-31-MTBF:1-25-02-1	220.00	b4e4cf6b61c9318f2b516053f5aa5ce302cd58b58130679ba972ebd4b028ccdc	28	1	LessonSharing	Lessonsharing - - - -1	
954	21	2026-08-03	Virement Lessonsharing 2026-07-31-MTBF:1-08-06-1	1520.00	ba7e0978bdcedeeea846dfc192f32ed6915e3e6e978fe4aa4ed953e8738e0ae2	28	1	LessonSharing	Lessonsharing - - - -1	
955	21	2026-08-03	Virement Lessonsharing 2026-07-31-MTBF:1-02-05-1	1240.00	45091ff873cfea95bf24152d2b112ebb65b43b094392a2cc4fc1ee821a27aaba	28	1	LessonSharing	Lessonsharing - - - -1	
957	21	2026-08-03	Carte X4519 Nyx*justeatemps Rois 31/07	-0.50	547dcee6724b972535f6f78332c78760caaa09ddbe0adacfb88bc28853fa43ee	28	1	Machine à café	Nyx Justeatemps Rois	
958	21	2026-08-03	Carte X4519 Nyx*justeatemps Rois 31/07	-1.00	12ec08b1d1b7e6efa34fb882c6ab445d5c051a42e1b4a09f8a0848852b030513	28	1	Machine à café	Nyx Justeatemps Rois	
959	21	2026-08-03	Vir Inst vers Ben-mohamed Madi T	-60.00	9a8ea47e6eaea9f4b75dc638676b14d7d6d2c0d2c027af06e897366a7301aba3	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
961	21	2026-08-04	X4519 Amazon Payments Pari 03/08	-34.90	c1096c028ac50d83622dec36292367bc59420e73bd69dd30be483627044bb298	26	1	Amazon	Amazon	
962	21	2026-08-04	Carte X4519 URSSAF Idf Montreuil 03/08	-1499.00	150df247cbdd7cb0d5d20f0a76156c8372984ee1c3f37a32db540b4787ef5a29	28	1	Cotisation URSSAF	URSSAF	tax
963	21	2026-08-04	Virement Vir Inst vers Aya N’goran	-50.00	e516391f133829d23a338796ff27a67f1d944a09b94599ae139f0122a777f873	33	1	Vers Sonia	Vers Aya N’goran	
964	21	2026-08-05	CAF De L Essonne 42036069V167990165TMADI TOUMANIO72026ME	181.43	2d1e4737cfb4d2cedbe1d7cf9e86a31e025f77acd6727a8ad5861fffa0074bab	30	1	Prestation CAF	CAF	benefit
966	21	2026-08-05	Prlv Free Telecom Free Hautdebit 149359473847939382 FR83277459654 Core	-29.99	808eacd0d29ecec408e9fe208fc2078f8eb359187580b68c5c1ebaacd642b1dd	22	1	Free	Free	
967	21	2026-08-07	Carte X4519 Deliveroo 75009 Pari 06/08	-16.29	e8e1ac88b8d65dbc81bb07f6c66d5b35e66f8e7611680fad0ba85a09fd3936ae	24	1	Deliveroo	Deliveroo	
968	21	2026-08-08	Virement Vir Inst vers Aya N’goran	-100.00	f038ef1758b25a91aa9141229af9bd3a3a17bb604eab2197561aacbbbafa355e	33	1	Vers Sonia	Vers Aya N’goran	
969	21	2026-08-09	Prlv Offre Essentiel	-6.00	4194141da42b1d4450b4b1461810379d02ac632c0200359b722b3e90c8e84266	30	1	Frais Bancaire	Offre Essentiel	
970	21	2026-08-10	Carte X4519 Paris 08/08	-21.55	1ffdadb1f0152152c2dcba355d16e57b74004f4a156f4ae55c1d1b85da53b1f9	28	1	Machine à café	Paris	
971	21	2026-08-10	X4519 Ubr”* Pending.uber.co 08/08	-13.94	735d9d5755818fbc45a182f0f1ab528aa521bad920e8a3778e8a05b6b042f649	23	1	Uber	Uber	
972	21	2026-08-10	Ret DAB X4519 Les Ulis 07/08 16H38	-20.00	9e46cd373a47837f767a6a49121e88d82d02fe13a2fe9612145b7e80adf69e94	24	1	Retrait Distributeur Les Ulis	Ret Les Ulis	
973	21	2026-08-10	Prlv Paypal (europe) $.a r.l. et Cie, 1052195188381 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-287.09	d56220bf0f64579872afc2330edba4cad47b106f6f95b2be905fb14983a93003	26	1	Paypal	PayPal	
975	21	2026-08-12	Vir Inst vers Ben-mohamed Madi T	-500.00	16d5cb4c970e50ad53ca098ea77fa39ea894c43ad863aca5e0a759d1ee880c0b	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
976	21	2026-08-14	Carte X4519 Dr Desmet Les Ulis 13/08	-30.00	4ff8ab41c36dd2e5236eef5d577b846c21a8adc8c7c9f013f426ab771d4bdd7d	24	1	Dr Desmet Les Ulis	Dr Desmet Les Ulis	
977	21	2026-08-14	X4519 Pharmacie Ulis 2 Les 13/08	-3.96	3ee50703627781c6b942d76996f616e40a9b92556743c0398aba53aff81108cd	25	1	Pharmacie Les Ulis	Pharmacie Ulis 2 Les	
978	21	2026-08-14	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-13.99	e0670b09281d90b400b1e44a0454bced1db73479db0336076b1487e94173220b	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
979	21	2026-08-17	X4519 Lmw*sncf Connect Par 15/08	-31.45	bd8ea469f8850b1f7f44582e4334d0bbb3bd71ae99068f21700e613880aca46e	23	1	SNCF	SNCF	
980	21	2026-08-17	X4519 Viandes Du 95 Argent 14/08	-75.15	d77c020ee5d124520974541e7c5f849fa1036caac482589684bad4a99de85a7e	24	1	Boucherie Argenteuil	Viandes Du Argent	
981	21	2026-08-17	Prlv Engie Mandat Core	-54.88	b080845b939dbdf83c17c9eaa9a71d5b9e9ee1a4a002598acf3421765f1659a8	22	1	Engie	Engie	
984	21	2026-08-19	Prlv Idfm IDFM/002980277-6292286-2980552/MAND ATE-REFERENCE000000000000556877 Core	-27.88	f9e11e2be0ed1bb1f8a003387b3b49d2207bf4a05f2faf9ef139dc46410fdbda	23	1	IDFM Core	Idfm Idfm - - Mand Ate- Core	
985	21	2026-08-20	Vir Inst vers Ben-mohamed Madi T	-63.00	0e996a49f836564aa164a1e252202ca63ea3ac4bbc1bb03d3d4e280dd174b0cd	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
986	21	2026-08-21	Vir Inst de M. Madi Tarik	800.00	4b3c2cd309884c3b757fdff39bbdc6101ab8f30e69734f298c984154f1e03216	29	1	De Tarik	De M. Madi Tarik	
987	21	2026-08-21	X4519 Lmw*sncf Connect Par 20/08	-28.00	dfdfb5481e51b9e3a3e336f73f26317a1f796e137841095a72ed66016fbbdb18	23	1	SNCF	SNCF	
982	21	2026-08-18	Carte X4519 Cofidis Amazon Ville 17/08	-63.68	c5075b05377129da263758785fbc64f37b19628c8fc420b68c0d350fac09b058	22	1	Amazon	Amazon	
483	12	2025-09-23	Carte X7303 Nyx*stcommerce Paris 22/09	-0.70	8e79f0a5010e48e5bcf07a37d3cf2e4d200ee8c79dc0922ffd40f03803eb90f4	28	1	Machine à café Paris	Nyx Stcommerce Paris	
965	21	2026-08-05	Prlv Acheel France REF-B1536EAD7309D7A041GFGWBH4 FR457Z787F911 Core	-12.47	af4cd87cc670c342916c2be98ac33b47a9688bcf9e7f0725942397c2b1f5f29b	22	1	Assurance habitation Acheel France	Acheel France Core	
983	21	2026-08-19	Vir Inst de Mr Attoumani Fazal E	2300.00	861a6f251a357de60f9586b47ebd34dd73cc2342602b097e3ee0df336f55fc2e	29	1	De Mr Attoumani Fazal E	De Mr Attoumani Fazal E	
\.


--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.users (id, username, password_hash, role, created_at) FROM stdin;
1	Ben	pbkdf2$sha256$210000$Nwkgc4OoYUlEzCP8YtTOgQ$wmavo1XXdxoUJ7tvZ+bHMk790ZPgza0rwaXfE2tP8ew	admin	2026-07-18 13:33:46.61631+00
2	localadm	pbkdf2$sha256$210000$RVTjzC7gHflUCBglzShg0Q$a3/fDxHvU27Djexp1eNZt2ZmZNgKNt7HYRyoo9MO5Uw	user	2026-07-18 17:42:32.761622+00
\.


--
-- Name: account_snapshots_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.account_snapshots_id_seq', 13, true);


--
-- Name: accounts_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.accounts_id_seq', 8, true);


--
-- Name: analyses_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.analyses_id_seq', 17, true);


--
-- Name: budgets_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.budgets_id_seq', 8, true);


--
-- Name: credits_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.credits_id_seq', 4, true);


--
-- Name: domains_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.domains_id_seq', 44, true);


--
-- Name: merchant_aliases_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.merchant_aliases_id_seq', 157, true);


--
-- Name: rules_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.rules_id_seq', 135, true);


--
-- Name: statements_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.statements_id_seq', 21, true);


--
-- Name: tags_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.tags_id_seq', 18384, true);


--
-- Name: taxes_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.taxes_id_seq', 6, true);


--
-- Name: transactions_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.transactions_id_seq', 988, true);


--
-- Name: users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.users_id_seq', 2, true);


--
-- Name: account_snapshots account_snapshots_account_id_snap_date_key; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.account_snapshots
    ADD CONSTRAINT account_snapshots_account_id_snap_date_key UNIQUE (account_id, snap_date);


--
-- Name: account_snapshots account_snapshots_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.account_snapshots
    ADD CONSTRAINT account_snapshots_pkey PRIMARY KEY (id);


--
-- Name: accounts accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_pkey PRIMARY KEY (id);


--
-- Name: analyses analyses_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.analyses
    ADD CONSTRAINT analyses_pkey PRIMARY KEY (id);


--
-- Name: budgets budgets_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.budgets
    ADD CONSTRAINT budgets_pkey PRIMARY KEY (id);


--
-- Name: credits credits_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.credits
    ADD CONSTRAINT credits_pkey PRIMARY KEY (id);


--
-- Name: domains domains_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.domains
    ADD CONSTRAINT domains_pkey PRIMARY KEY (id);


--
-- Name: merchant_aliases merchant_aliases_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.merchant_aliases
    ADD CONSTRAINT merchant_aliases_pkey PRIMARY KEY (id);


--
-- Name: rules rules_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.rules
    ADD CONSTRAINT rules_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (name);


--
-- Name: sessions sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (token);


--
-- Name: statements statements_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.statements
    ADD CONSTRAINT statements_pkey PRIMARY KEY (id);


--
-- Name: tag_folder_prefs tag_folder_prefs_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.tag_folder_prefs
    ADD CONSTRAINT tag_folder_prefs_pkey PRIMARY KEY (owner_id, tag_name);


--
-- Name: tags tags_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.tags
    ADD CONSTRAINT tags_pkey PRIMARY KEY (id);


--
-- Name: taxes taxes_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.taxes
    ADD CONSTRAINT taxes_pkey PRIMARY KEY (id);


--
-- Name: transaction_tags transaction_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.transaction_tags
    ADD CONSTRAINT transaction_tags_pkey PRIMARY KEY (transaction_id, tag_id);


--
-- Name: transactions transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: users users_username_key; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_username_key UNIQUE (username);


--
-- Name: idx_accounts_owner; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_accounts_owner ON public.accounts USING btree (owner_id);


--
-- Name: idx_accounts_owner_name; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_accounts_owner_name ON public.accounts USING btree (owner_id, name);


--
-- Name: idx_analyses_owner_month; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_analyses_owner_month ON public.analyses USING btree (owner_id, month);


--
-- Name: idx_budgets_owner_domain; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_budgets_owner_domain ON public.budgets USING btree (owner_id, domain_id);


--
-- Name: idx_domains_owner; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_domains_owner ON public.domains USING btree (owner_id);


--
-- Name: idx_domains_owner_name_parent; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_domains_owner_name_parent ON public.domains USING btree (owner_id, name, COALESCE(parent_id, (0)::bigint));


--
-- Name: idx_domains_owner_root_name; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_domains_owner_root_name ON public.domains USING btree (owner_id, name) WHERE (parent_id IS NULL);


--
-- Name: idx_merchant_aliases_owner_key; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_merchant_aliases_owner_key ON public.merchant_aliases USING btree (owner_id, lower(match_key));


--
-- Name: idx_rules_owner; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_rules_owner ON public.rules USING btree (owner_id);


--
-- Name: idx_rules_owner_pattern_ci; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_rules_owner_pattern_ci ON public.rules USING btree (owner_id, lower(pattern));


--
-- Name: idx_sessions_expires; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_sessions_expires ON public.sessions USING btree (expires_at);


--
-- Name: idx_settings_owner_key; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_settings_owner_key ON public.settings USING btree (COALESCE(owner_id, (0)::bigint), key);


--
-- Name: idx_statements_owner; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_statements_owner ON public.statements USING btree (owner_id);


--
-- Name: idx_tags_owner; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_tags_owner ON public.tags USING btree (owner_id);


--
-- Name: idx_tags_owner_name; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_tags_owner_name ON public.tags USING btree (owner_id, name);


--
-- Name: idx_transactions_label; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_transactions_label ON public.transactions USING gin (to_tsvector('simple'::regconfig, label));


--
-- Name: idx_transactions_owner; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_transactions_owner ON public.transactions USING btree (owner_id);


--
-- Name: idx_transactions_owner_hash; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_transactions_owner_hash ON public.transactions USING btree (owner_id, hash);


--
-- Name: idx_transactions_owner_merchant; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_transactions_owner_merchant ON public.transactions USING btree (owner_id, merchant);


--
-- Name: idx_tx_date; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_tx_date ON public.transactions USING btree (tx_date);


--
-- Name: account_snapshots account_snapshots_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.account_snapshots
    ADD CONSTRAINT account_snapshots_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id) ON DELETE CASCADE;


--
-- Name: accounts accounts_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: analyses analyses_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.analyses
    ADD CONSTRAINT analyses_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: budgets budgets_domain_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.budgets
    ADD CONSTRAINT budgets_domain_id_fkey FOREIGN KEY (domain_id) REFERENCES public.domains(id) ON DELETE CASCADE;


--
-- Name: budgets budgets_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.budgets
    ADD CONSTRAINT budgets_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: credits credits_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.credits
    ADD CONSTRAINT credits_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: domains domains_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.domains
    ADD CONSTRAINT domains_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: domains domains_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.domains
    ADD CONSTRAINT domains_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.domains(id) ON DELETE CASCADE;


--
-- Name: merchant_aliases merchant_aliases_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.merchant_aliases
    ADD CONSTRAINT merchant_aliases_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: rules rules_domain_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.rules
    ADD CONSTRAINT rules_domain_id_fkey FOREIGN KEY (domain_id) REFERENCES public.domains(id) ON DELETE CASCADE;


--
-- Name: rules rules_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.rules
    ADD CONSTRAINT rules_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: sessions sessions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: settings settings_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.settings
    ADD CONSTRAINT settings_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: statements statements_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.statements
    ADD CONSTRAINT statements_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id);


--
-- Name: statements statements_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.statements
    ADD CONSTRAINT statements_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: tag_folder_prefs tag_folder_prefs_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.tag_folder_prefs
    ADD CONSTRAINT tag_folder_prefs_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: tags tags_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.tags
    ADD CONSTRAINT tags_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: taxes taxes_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.taxes
    ADD CONSTRAINT taxes_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: transaction_tags transaction_tags_tag_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.transaction_tags
    ADD CONSTRAINT transaction_tags_tag_id_fkey FOREIGN KEY (tag_id) REFERENCES public.tags(id) ON DELETE CASCADE;


--
-- Name: transaction_tags transaction_tags_transaction_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.transaction_tags
    ADD CONSTRAINT transaction_tags_transaction_id_fkey FOREIGN KEY (transaction_id) REFERENCES public.transactions(id) ON DELETE CASCADE;


--
-- Name: transactions transactions_domain_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_domain_id_fkey FOREIGN KEY (domain_id) REFERENCES public.domains(id) ON DELETE SET NULL;


--
-- Name: transactions transactions_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: transactions transactions_statement_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_statement_id_fkey FOREIGN KEY (statement_id) REFERENCES public.statements(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict JkGxOZXZp4jBtqLWMScW9wdosqiab0oTLgO948iqL4WDZjOdVPoeoipRIhyGej8

