# make-qr-code

Utilitaire en ligne de commande en Go : texte ou URL + logo PNG/JPEG → QR-Code PNG **2048 × 2048 pixels**. Fonctionne en local, sans API ni abonnement.

## Compiler

Go 1.24 ou supérieur. Le premier téléchargement des dépendances nécessite Internet ; le programme compilé fonctionne hors ligne, sans CGO.

```sh
go build -o make-qr-code .
```

## Utilisation

```sh
./make-qr-code 'https://example.org/formulaire' logo.png
./make-qr-code -o mon-qr.png 'Bonjour tout le monde !' logo.jpg
./make-qr-code -text 'https://example.org/?a=1&b=2' -logo logo.png -o mon-qr.png
./make-qr-code -h
```

Le fichier par défaut est `qrcode.png`, dans le dossier courant. Placer les options **avant** les arguments positionnels. Pour un texte commençant par `-`, employer `-text` ou placer `--` avant les deux arguments. Entourer le texte de guillemets pour préserver les espaces et caractères spéciaux du shell.

Les deux paramètres sont obligatoires. Le contenu est encodé tel quel en UTF-8 : les URL ne sont ni modifiées ni raccourcies. Un contenu trop volumineux est refusé avec un message d'erreur.

## Logo et lisibilité

- PNG (transparence acceptée) ou JPEG, détecté d'après le contenu du fichier ; maximum 40 millions de pixels.
- Logo centré, proportions conservées, redimensionnement lissé et fond blanc avec marge.
- QR noir sur blanc, modules entiers sans flou et zone blanche extérieure d'au moins quatre modules.
- Correction d'erreurs H. Le logo occupe initialement 18 % de la largeur du QR hors marge ; si nécessaire, il est réduit par paliers jusqu'à 10 %.
- Le PNG final est redécodé automatiquement et son contenu comparé exactement au texte fourni. En cas d'échec, aucun fichier n'est enregistré. Ce contrôle logiciel ne garantit pas la lecture par tous les appareils : tester le QR à sa taille d'impression ou d'affichage finale.

Le dossier de sortie doit exister. Un fichier de sortie existant est remplacé après génération et validation, via un fichier temporaire dans le même dossier. Le logo source ne peut pas servir de fichier de sortie. Une erreur produit un code de sortie non nul.

## Vérifier et compiler pour d'autres plateformes

```sh
go test ./...
go vet ./...
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o dist/make-qr-code-macos-arm64 .
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o dist/make-qr-code-macos-amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/make-qr-code-linux-amd64 .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/make-qr-code-windows-amd64.exe .
```

## Bibliothèques

- [skip2/go-qrcode](https://github.com/skip2/go-qrcode) : encodage QR.
- [makiuchi-d/gozxing](https://github.com/makiuchi-d/gozxing) : contrôle du décodage.
- [golang.org/x/image](https://pkg.go.dev/golang.org/x/image/draw) : redimensionnement du logo.

Les versions sont fixées dans `go.mod` et `go.sum`.
