# Lire et écrire `~/.ssh/config` comme OpenSSH, sans perdre de clés

> Statut : validé par Gu1llaum-3 le 03/10/2026
> Origine : audit de la PR 59 du 03/10/2026 (bugs hors PR), conversation du 03/10/2026
> Base : `dev` (la branche de travail part de `dev`, et la PR revient sur `dev`)

## Objectif

Deux bugs de sshm abîment le fichier de config de l'utilisateur.

1. **Les clés en plus disparaissent.** Un hôte qui a plusieurs lignes `IdentityFile` n'en garde
   qu'une au parsing. L'enregistrer depuis le formulaire d'édition, même sans rien changer,
   supprime donc les autres clés de `~/.ssh/config`, en silence.
2. **Le découpage des lignes ne suit pas OpenSSH.** sshm découpe les lignes avec
   `strings.Fields`, puis retire des guillemets au cas par cas. Résultats :
   - les guillemets simples et les antislashs ne sont pas compris ;
   - les espaces multiples sont écrasés ;
   - les commentaires en fin de ligne restent dans la valeur ;
   - `Include "…"` avec des espaces est ignoré ;
   - des guillemets restent dans `User`, `HostName` et `ProxyJump` ;
   - un bloc écrit `host foo`, `Host<tab>foo`, `Host=foo` ou `Host "foo"` est lu, mais ne peut
     être ni modifié ni supprimé (« host not found ») ;
   - `ProxyCommand=…`, que sshm écrit lui-même sous cette forme, ne fait pas l'aller-retour.

Après ce plan, sshm lit chaque ligne comme `ssh` la lit, réécrit les valeurs de façon à ce
qu'`ssh` et sshm les relisent à l'identique, et conserve toutes les clés d'un hôte, dans leur
ordre.

## Décisions

- **Un seul découpeur, calqué sur OpenSSH,** dans un nouveau fichier `internal/config/tokenize.go`.
  Il est utilisé par tout ce qui lit une ligne de config : parseur, `Include`, recherche rapide
  et fonctions qui retrouvent un bloc (modifier, supprimer, déplacer, détecter un bloc à
  plusieurs hôtes). Décidé le 03/10/2026. Règles relevées avec `ssh -G` (OpenSSH 10.3) :
  - les espaces et les tabulations séparent les mots, et le mot-clé peut être suivi de `=`,
    avec ou sans espaces ;
  - les guillemets doubles et simples regroupent, et des morceaux collés se concatènent
    (`"/a"b` donne `/ab`) ;
  - entre guillemets doubles, `\"` donne `"` et `\ ` reste `\ ` ;
  - entre guillemets simples, tout reste littéral ;
  - hors guillemets, `\` n'échappe que l'espace, `"`, `'` et `\`. Tout autre antislash est
    gardé, donc `C:\Users\me\id` reste intact ;
  - un `#` en début de mot ouvre un commentaire jusqu'à la fin de la ligne. Un `#` au milieu
    d'un mot, ou entre guillemets, est littéral.
- **Trois sortes de valeurs :**
  - **brutes** (`ProxyCommand`, `RemoteCommand`, `LocalCommand`, `KnownHostsCommand`) : le reste
    de la ligne tel quel, commentaires compris, comme le fait OpenSSH ;
  - **listes** (`Host`, `Include`) : un mot par motif ou par chemin ;
  - **simples** (toutes les autres).
- **sshm reste tolérant là où OpenSSH refuse la ligne.**
  - **Valeur simple en plusieurs mots :** les mots sont recollés par un espace. C'est le
    comportement d'aujourd'hui, qui répare `IdentityFile C:\My Drive\key` écrit sans guillemets
    (le cas de l'issue #16) : l'écriture remet ensuite les guillemets.
  - **Guillemet non fermé :** le reste de la ligne est pris tel quel, sans erreur.
- **Les directives inconnues sont rangées dans `Options` avec leurs arguments bruts**
  (guillemets compris, commentaire retiré), au lieu de mots recollés. L'aller-retour sur le
  disque est ainsi exact.
- **Plusieurs `IdentityFile` :** `Identity` garde la **première** clé, celle qu'OpenSSH essaie
  en premier (décidé le 03/10/2026). Les suivantes vont dans un nouveau champ
  `ExtraIdentities []string`, dans l'ordre du fichier. Elles sont réécrites juste après la ligne
  `IdentityFile` principale, et affichées en lecture seule dans le formulaire d'édition et la
  vue info.
- **Refactorisation préalable : une seule fonction écrit les directives d'un hôte.**
  Aujourd'hui, l'ordre d'écriture est recopié dans `AddSSHHostToFile` et dans six branches de
  `UpdateSSHHostInFile` et `UpdateMultiHostBlock`. Sans cette fonction commune, chaque
  correctif devrait être appliqué sept fois.
- **L'écriture est symétrique de la lecture.** `formatSSHConfigValue` met des guillemets dès que
  le découpeur couperait ou transformerait la valeur (espace, tabulation, `#` en tête,
  guillemet, antislash suivi d'un caractère spécial), et échappe ce qu'il faut. La règle est
  appliquée à `IdentityFile`, aux clés en plus, à `User`, `HostName` et `ProxyJump`, ainsi
  qu'aux noms de `Host`. `ProxyCommand` et `RemoteCommand` restent bruts. Le test existant
  `ssh_test.go:1508`, qui valide aujourd'hui une sortie fausse, est corrigé.
- **`ssh -G` sert d'oracle dans les tests.** Un test compare le découpeur et l'aller-retour
  d'écriture à `ssh -G -F <fichier>`. Il est sauté (`t.Skip`) si `ssh` est absent.
- **Pas de recherche externe :** le comportement de référence est relevé avec `ssh -G` sur la
  machine, et le dépôt a déjà des modèles de tests (tests en tableau, allers-retours sur des
  blocs à plusieurs hôtes).

## Critères d'acceptation

- AC-1 : le découpeur donne, pour chaque cas de la table ci-dessous, le même résultat que
  `ssh -G`. Le test oracle passe quand `ssh` est installé.

  | Ligne | Valeur attendue |
  |---|---|
  | `IdentityFile "/a  b/id"` | `/a  b/id` |
  | `IdentityFile '/a b/id'` | `/a b/id` |
  | `IdentityFile /a\ b/id` | `/a b/id` |
  | `IdentityFile C:\Users\me\id` (avec ou sans guillemets) | `C:\Users\me\id` |
  | `IdentityFile C:\\x\\id` | `C:\x\id` |
  | `IdentityFile "/a\"b"` | `/a"b` |
  | `IdentityFile '/a\ b'` / `"/a\ b"` | `/a\ b` |
  | `IdentityFile "/a"b` | `/ab` |
  | `IdentityFile /a/id # old key` | `/a/id` |
  | `IdentityFile /a/id#x` / `"/a #b"` | `/a/id#x` / `/a #b` |
  | `IdentityFile=/a/id`, `IdentityFile = /a/id`, tabulations | `/a/id` |
  | `User "john doe"`, `HostName "ex.com"` | `john doe`, `ex.com` |
  | `ProxyCommand ssh -W "%h:%p" bastion # c` | la ligne brute après le mot-clé |

- AC-2 : `Include "<dossier avec espace>/*"` charge les hôtes inclus, et `Include a b` traite
  deux motifs.
- AC-3 : un hôte écrit par sshm avec `ProxyCommand` (forme `ProxyCommand=…`), y compris
  `ProxyCommand=none`, est relu dans `ProxyCommand`. Il ne se retrouve ni dans `Options`, ni
  perdu.
- AC-4 : les cas tolérés ne plantent pas et suivent la décision : `IdentityFile C:\My Drive\key`
  sans guillemets donne `C:\My Drive\key`, et un guillemet non fermé donne le reste de la ligne.
- AC-5 : un hôte avec `IdentityFile ~/.ssh/a`, puis `IdentityFile "~/.ssh/b c"`, puis
  `ForwardAgent yes` est lu avec `Identity="~/.ssh/a"` et `ExtraIdentities=["~/.ssh/b c"]`.
  L'enregistrer sans changement par `UpdateSSHHostInFile` (bloc simple) et par
  `UpdateMultiHostBlock` (bloc à plusieurs hôtes) laisse les trois lignes dans le fichier, dans
  le même ordre. Le déplacement d'un fichier à l'autre (`MoveHostToFile`) les conserve aussi.
- AC-6 : le formulaire d'édition et la vue info affichent les clés en plus, en lecture seule.
  Modifier la clé principale dans le formulaire ne touche pas les clés en plus.
- AC-7 : un bloc écrit `host foo`, `Host<tab>foo`, `Host=foo` ou `Host "foo"` peut être modifié
  et supprimé. Avec deux hôtes du même nom dans deux fichiers, la modification et la
  suppression visent toujours le bon bloc (`LineNumber`).
- AC-8 : pour un jeu de valeurs piégées (espace, deux espaces, tabulation, `"`, `'`, `\`, `#` en
  tête, chemin Windows, chemin Windows avec espace), écrire la valeur puis la relire donne la
  même valeur, avec sshm comme avec `ssh -G`. Cela vaut pour `IdentityFile`, les clés en plus,
  `User`, `HostName` et `ProxyJump`.
- AC-9 : `go build`, `go vet` et toute la suite de tests passent. Les tests existants ne
  changent pas, sauf l'attente fausse de `ssh_test.go:1508`.

## Interfaces à tester

- `config.ParseSSHConfigFile` : AC-1 à AC-5.
- Le découpeur (fonction interne au paquet `config`), en test de tableau dans le même paquet,
  et comparé à `ssh -G` : AC-1, AC-8.
- `config.AddSSHHostToFile`, `UpdateSSHHostInFile`, `UpdateMultiHostBlock`, `MoveHostToFile`,
  `DeleteSSHHostFromFileWithLine` : AC-3, AC-5, AC-7, AC-8.
- `formatSSHConfigValue` (table existante) : AC-8.
- Formulaire d'édition et vue info : AC-6. Il n'existe pas de test d'interface dans le dépôt,
  donc ce critère passe par une vérification manuelle.

## Hors périmètre

- Le champ Options du formulaire, découpé sur la sous-chaîne `-o`
  (`ParseSSHOptionsFromCommand`, `ssh.go:684-725`), qui casse sur `~/.ssh/id-other` et sur `=`.
  Une issue à part.
- Pour une clé répétée dans un bloc, sshm garde la dernière valeur alors qu'OpenSSH garde la
  première. `IdentityFile` est traité ici, mais pas les autres clés.
- La sauvegarde de config, qui ne garde qu'une version (`backupConfig`, `ssh.go:129`).
- `Match`, `SendEnv` et `SetEnv` comme listes : ils restent dans `Options`, avec leurs arguments
  bruts.
- Les défauts du TUI vus pendant l'audit de la PR 60 (curseur après `H`, complétion des hôtes
  cachés).

## Questions ouvertes

- **[Mainteneur]** Faut-il exposer les clés en plus dans le JSON de `sshm info`
  (`sshm.info.v1`) et de `sshm search` ? Hypothèse : non, dans ce plan. Le JSON ne change pas
  de forme. Mais `identity_file` d'un hôte à plusieurs clés renvoie désormais la **première**
  clé au lieu de la dernière, et cette différence sera notée dans la PR.

## Vague 1 : le découpeur et la lecture

- [x] 1.1 Test qui reproduit les bugs de lecture et échoue : table AC-1 sur
  `ParseSSHConfigFile`, plus `Include` avec espace (AC-2) et `ProxyCommand=` relu (AC-3). Il
  reste comme test de non-régression. (AC-1, AC-2, AC-3)
- [x] 1.2 Le découpeur `tokenize.go` et sa table de tests, en TDD, avec le test oracle `ssh -G`
  sauté si `ssh` est absent. (AC-1, AC-4)
- [x] 1.3 Brancher le découpeur dans le parseur principal : mot-clé avec `=`, trois sortes de
  valeurs, `Options` avec arguments bruts. Retirer les `strings.Trim` au cas par cas (`Host`,
  `IdentityFile`). Le test 1.1 passe. (AC-1, AC-3, AC-4)
- [x] 1.4 Brancher le découpeur dans `processIncludeDirective`, `quickHostSearchInFile` et
  `quickSearchInclude`. (AC-2)

## Vague 2 : ne plus perdre de clés

- [x] 2.1 Refactorisation préalable : une seule fonction écrit les directives d'un hôte, dans
  l'ordre actuel. Elle est utilisée par `AddSSHHostToFile` et par les six branches de
  `UpdateSSHHostInFile` et `UpdateMultiHostBlock`. Aucun comportement ne change, et les tests
  existants restent verts. (AC-9)
- [x] 2.2 Test qui reproduit la perte de clés et échoue : un hôte à deux `IdentityFile` plus
  `ForwardAgent`, enregistré sans changement par les deux chemins de sauvegarde, puis déplacé.
  (AC-5)
- [x] 2.3 `ExtraIdentities` : la première clé dans `Identity`, les suivantes dans l'ordre,
  réécrites après la clé principale par la fonction de 2.1. Le test 2.2 passe. (AC-5)
- [x] 2.4 Formulaire d'édition et vue info : les clés en plus en lecture seule, et
  `submitEditForm` les transmet sans les modifier. Vérification prévue : `go run . -c <config
  de test>`, avec une capture du formulaire et de la vue info pour un hôte à deux clés, puis
  l'enregistrement et la comparaison du fichier. (AC-6)

## Vague 3 : retrouver les blocs et écrire symétriquement

- [x] 3.1 Test qui reproduit « host not found » et échoue : modification et suppression de
  blocs `host foo`, `Host<tab>foo`, `Host=foo` et `Host "foo"`, plus le cas de deux hôtes du
  même nom dans deux fichiers. (AC-7)
- [ ] 3.2 Les fonctions qui retrouvent un bloc (`UpdateSSHHostInFile`, `UpdateMultiHostBlock`,
  `DeleteSSHHostFromFileWithLine`, `HostExistsInSpecificFile`, `IsPartOfMultiHostDeclaration`)
  reconnaissent les lignes `Host` avec le découpeur. Le test 3.1 passe. (AC-7)
- [ ] 3.3 Écriture symétrique : `formatSSHConfigValue` met des guillemets et échappe selon les
  règles du découpeur. La règle est appliquée par la fonction de 2.1 à `IdentityFile`, aux clés
  en plus, à `User`, `HostName`, `ProxyJump` et aux noms de `Host`. L'attente fausse de
  `ssh_test.go:1508` est corrigée. Les tests d'aller-retour avec sshm et avec `ssh -G` passent.
  (AC-8)
- [ ] 3.4 Vérification d'ensemble : `go build`, `go vet` et `go test ./...` ; tests oracle lancés
  avec `ssh` présent ; essai manuel du TUI sur une copie de config réelle (ajout, modification,
  déplacement et suppression d'un hôte à deux clés et d'un hôte aux valeurs entre guillemets),
  avec comparaison du fichier avant et après. (AC-9)

## Journal

- 03/10/2026, vague 1 terminée (commits `8ad2763`, `0179635`, `6ae676f`, `93f8827`, branche
  `fix/openssh-config-parsing` partie de `dev`).
  - **Livré :**
    - `internal/config/tokenize.go` : `splitConfigLine` (mot-clé, arguments, texte brut avec et
      sans commentaire) et `isVerbatimKeyword` ;
    - le parseur principal et la recherche rapide (`quickHostSearchInFile`) utilisent le
      découpeur ;
    - `Include` suit chaque argument comme un motif ;
    - `ProxyCommand` et `RemoteCommand` gardent la ligne brute ;
    - les directives inconnues gardent leurs arguments tels qu'écrits ;
    - les `strings.Trim` au cas par cas (`Host`, `IdentityFile`, recherche rapide) sont retirés.
  - **Tests :**
    - `openssh_syntax_test.go` : AC-1 (19 cas), AC-2 (parseur et recherche rapide), AC-3,
      AC-4 au niveau du parseur ;
    - `tokenize_test.go` : table du découpeur, texte brut, lignes ignorées, cas tolérés, et le
      test oracle qui compare avec `ssh -G`. Il a tourné avec OpenSSH 10.3 et n'a pas été
      sauté.
  - **Écarts au plan :**
    - AC-2 est passé dès la tâche 1.3, parce que c'est le parseur principal qui découpe la
      ligne `Include`. La tâche 1.4 a donc surtout porté sur la recherche rapide, avec son
      propre test (`Host=…`, noms entre guillemets, `#` en fin de ligne) ;
    - `processIncludeDirective` n'a pas eu besoin de changer : il reçoit désormais un seul motif
      déjà découpé.
  - **Vérifié :** `go build`, `go vet` et `go test ./...`, code de retour 0. Le test 1.1, rouge
    au commit `8ad2763`, est vert depuis `6ae676f`.
  - **Non vérifié :** le comportement sur une vraie config. C'est l'essai proposé ci-dessous.
  - **Trouvé en route :** `gofmt` signale `internal/config/appconfig.go` et
    `appconfig_test.go`, mal formatés avant ce travail. Hors plan, je n'y ai pas touché.
  - **Quoi essayer :** `go run .` sur ta vraie config. La liste des hôtes doit être la même
    qu'avant. Regarde en particulier les hôtes déclarés dans des fichiers inclus, et un hôte avec
    un `ProxyCommand` ou un `IdentityFile` entre guillemets (vue info avec `i`). Puis
    `go run . <un hôte>` doit se connecter comme avant.
  - **Retour d'essai (Gu1llaum-3, 03/10/2026) :** essai fait sur la vraie config, « ça
    fonctionne ». Feu vert pour la vague 2.
- 03/10/2026, vague 2 terminée (commits `5bada86`, `975db56`, `80c72d6`, `0348095`).
  - **Livré :**
    - `hostDirectiveLines` (`ssh.go`) écrit les directives d'un hôte pour `AddSSHHostToFile` et
      les six branches de mise à jour (−259 lignes de copies) ;
    - le champ `SSHHost.ExtraIdentities` : la première `IdentityFile` va dans `Identity`, les
      suivantes dans ce champ, dans l'ordre. Elles sont réécrites juste après la clé principale ;
    - le formulaire d'édition affiche chaque clé en plus sous « Identity File », avec la
      mention « (read-only, kept on save) », et les transmet à l'enregistrement. La vue info
      liste toutes les clés.
  - **Tests :**
    - `internal/config/identity_files_test.go` : AC-5 en lecture, et en sauvegarde par bloc
      simple, par bloc à plusieurs hôtes et par déplacement (`HOME` temporaire). Rouge au commit
      `975db56` (la première clé disparaissait sur les trois chemins), vert depuis `80c72d6` ;
    - `internal/ui/identity_files_test.go` : AC-6, le formulaire garde la clé en plus quand on
      change la clé principale, et les deux vues l'affichent.
  - **Écarts au plan :**
    - AC-6 est aussi couvert par des tests automatiques sur le modèle du formulaire, en plus de
      l'essai manuel prévu. Le plan supposait qu'aucun test d'interface n'était possible ;
    - `Identity` garde désormais la première clé. Si l'utilisateur vide le champ « Identity
      File », la première clé en plus devient la clé principale à la lecture suivante.
  - **Vérifié :**
    - `go build`, `go vet` et `go test ./...`, code de retour 0 ;
    - essai réel dans le TUI (tmux, `-c` sur une config de test, `HOME` isolé) : la vue info
      montre les deux clés, le formulaire montre la clé en plus en lecture seule, et après
      Ctrl+S le fichier est identique à l'original (`diff` vide).
  - **Non vérifié :** l'enregistrement est refusé si la clé principale n'existe pas sur le
    disque. C'est la validation existante (`validation.ValidateHost`), inchangée. Les clés en
    plus ne sont pas validées.
  - **Trouvé en route :** `internal/ui/file_selector.go` ne passe pas `gofmt` (antérieur, noté
    hors plan).
  - **Quoi essayer :** sur une copie de ta config, avec un hôte qui a deux `IdentityFile` :
    `cp ~/.ssh/config /tmp/c && go run . -c /tmp/c`. Puis :
    - `i` sur cet hôte : les deux clés apparaissent ;
    - `e` : la seconde clé est listée sous « Identity File » ;
    - Ctrl+S sans rien changer, puis `diff ~/.ssh/config /tmp/c` : rien ne doit avoir disparu.

## Hors plan

- `internal/config/appconfig.go` et `appconfig_test.go` ne passent pas `gofmt` (antérieur).
- `internal/ui/file_selector.go` ne passe pas `gofmt` non plus (antérieur).
