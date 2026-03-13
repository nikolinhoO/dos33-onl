1. Выполните команды, что они выводят, как можно использовать вывод упомянутых команды

### Информация о релизе ОС
```console
    cat /etc/os-release
    lsb_release -a
```
![alt text](OS_release-1.png)

### Информация об архитектуре процессора, а также ядре, ОС и другом оборудовании
```console
    uname -m
    uname -a
```
![alt text](uname-1.png)
### Информация об оперативной памяти
```console
    free -h
    cat /proc/meminfo | grep MemTotal
```
![alt text](meminfo-1.png)
### Информация о процессоре
```console
    lscpu
    cat /proc/cpuinfo | grep "model name"
```
![alt text](lscpu-1.png)