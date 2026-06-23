YC2_NAME=( $(yc compute instance list --format json | jq -r '.[].id') )

echo "Список YC2, которые будут удалены:"
for item in "${YC2_NAME[@]}"; do
        echo "$item"
done

echo -e "\nПодождите, происходит очистка и удаление YC2..."

for del_yc2 in "${YC2_NAME[@]}"; do
        echo "Удаление YC2: $del_yc2"
        yc compute instance delete $del_yc2
done

echo -e "\nУдаление прошло успешно."