# antlr-generate.ps1

# Путь к JAR-файлу ANTLR (предполагается, что файл в текущей директории)
$antlrJar = ".\antlr-4.13.2-complete.jar"

# Проверка наличия JAR-файла
if (-Not (Test-Path $antlrJar)) {
    Write-Error "Файл $antlrJar не найден. Убедитесь, что он существует в текущей директории."
    exit 1
}

# Генерация Go-парсера из всех .g4 файлов в текущей директории
# -Dlanguage=Go: генерировать для Go
# -no-visitor: не создавать Visitor (если нужен — убери этот флаг)
# -package parsing: задать пакет
# *.g4: применить ко всем .g4
java -Xmx500M -cp "$antlrJar" org.antlr.v4.Tool -Dlanguage=Go -no-visitor -package parsing *.g4
