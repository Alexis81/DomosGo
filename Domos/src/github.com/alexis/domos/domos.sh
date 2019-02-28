stty -F /dev/ttyS0 9600 clocal cread cs8 -cstopb -parenb

echo "Lancement de l'application Domos..."
go run *.go
