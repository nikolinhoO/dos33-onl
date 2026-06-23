BUCKET_NAME=( $(yc storage bucket list --format json | jq -r '.[].name') )

echo "Список S3, которые будут очищены и удалены:"
for item in "${BUCKET_NAME[@]}"; do
        echo "$item"
done

echo -e "\nПодождите, происходит очистка и удаление S3..."


for del_obj in "${BUCKET_NAME[@]}"; do
        echo "Обработка S3: $del_obj"
        yc storage s3 rm s3://$del_obj --recursive
        yc storage bucket delete --name $del_obj
done

echo -e "\nУдаление прошло успешно"