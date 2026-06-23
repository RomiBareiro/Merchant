# Orchestration API

## Objetivo

El objetivo de esta tarea es construir una API para orquestrar la creación de transacciones y cuentas por cobrar (como un servicio de pago).
Es importante asegurar la consistencia de los datos.


### Tarea: Crear API de Orquestración

El objetivo de esta tarea es construir una API que procese y almacene transacciones iniciadas por un merchant (comerciante), asegurando la deducción correcta
de comisiones y la creación de las cuentas por cobrar (receivables) correspondientes.

Una transacción creada debe incluir:

- Un ID único creado usando la Numerator API
- El monto total de la transacción, formateado como una cadena decimal.
- Una descripción de la transacción, por ejemplo, "Remera Negra M".
- Método de pago: **debit_card** o **credit_card**.
- El número de tarjeta (solo los últimos 4 dígitos deben ser almacenados y devueltos, ya que es información sensible).
- El nombre del titular de la tarjeta.
- Fecha de vencimiento de la tarjeta en formato MM/AA.
- CVV de la tarjeta.

Al crear una transacción, **una cuenta por cobrar del merchant también debe ser creada**, una cuenta por cobrar representa la porción
del monto de la transacción que va al merchant después de deducir la comisión aplicable.

#### Reglas para Crear Cuentas por Cobrar

| Tipo de Transacción | Estado de la Cuenta por Cobrar | Fecha de Pago                        | Comisión |
| ------------------- | ------------------------------ | ------------------------------------ | -------- |
| **Debit Card**      | `paid`                         | Misma fecha de creación (D + 0)      | 2%       |
| **Credit Card**     | `waiting_funds`                | Fecha de creación + 30 días (D + 30) | 4%       |

**Ejemplo**: Si una cuenta por cobrar es creada con un valor de ARS 100,00 de una transacción con **credit_card**, el
merchant recibirá ARS 96,00 (la comisión se calcula basada en el monto total de la transacción).

### Generación de ID Único con Numerator API

Es esencial que **_transacciones y cuentas por cobrar_** tengan IDs únicos generados. El **Numerator Service**
simula un sistema externo que te ayuda a implementar tu propia lógica de generación de ID.


## Configuración

### Iniciar servicios proporcionados

```
docker compose up
```

Esto expondrá:

1. En http://0.0.0.0:8080/ la API para gestionar transacciones y cuentas por cobrar
2. En http://0.0.0.0:3000/ la API para generación de IDs.

## Resumen de Servicios de la API

### Transacciones

| Endpoint           | Método   | Descripción                                                              | Cuerpo de la Solicitud                                                                                                                                                                                  |
| ------------------ | -------- | ------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `transactions`     | `GET`    | Listar todas las transacciones.                                          | -                                                                                                                                                                                                       |
| `transactions/:id` | `GET`    | Obtener detalles de una transacción específica por ID.                   | -                                                                                                                                                                                                       |
| `transactions`     | `POST`   | Crear una nueva transacción. Usa Numerator API para generar un ID único. | `{ "id": <string>, "value": "250.00", "description": "T-Shirt", "method": "credit_card", "cardNumber": "2222", "cardHolderName": "Simplenube Store", "cardExpirationDate": "04/28", "cardCvv": "222" }` |
| `transactions/:id` | `DELETE` | Eliminar una transacción por ID.                                         | -                                                                                                                                                                                                       |

### Cuentas por Cobrar

| Endpoint          | Método   | Descripción                                                                    | Cuerpo de la Solicitud                                                                                                                                                           |
| ----------------- | -------- | ------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `receivables`     | `GET`    | Listar todas las cuentas por cobrar.                                           | -                                                                                                                                                                                |
| `receivables/:id` | `GET`    | Obtener detalles de una cuenta por cobrar específica por ID.                   | -                                                                                                                                                                                |
| `receivables`     | `POST`   | Crear una nueva cuenta por cobrar. Usa Numerator API para generar un ID único. | `{ "id": <string>, "transaction_id": <string>, "status": "waiting_funds", "create_date": "2022-05-20T19:20:14.576-03:00", "payment_date": "2022-06-19T19:20:14.576-03:00", "subtotal": "250.00", "discount": "10.00", "total": "240.00" }` |
| `receivables/:id` | `DELETE` | Eliminar una cuenta por cobrar por ID.                                         | -                                                                                                                                                                                |

Si la transacción ya fue persistida pero falla la creación del receivable, la API de orquestación intenta borrar la transacción como compensación best-effort. No es una transacción ACID entre servicios HTTP, por lo que las fallas de rollback se registran en logs y se devuelven en el error.

### Numerator

Numerator Service es un servicio que proporciona el ID actual y almacena el siguiente, que son requeridos para crear transacciones y cuentas por cobrar.
Aunque el servicio ofrece varios endpoints, no estás obligado a usar todos ellos. La implementación puede hacerse de varias maneras dependiendo de tu enfoque
para generar IDs únicos.

| Endpoint                 | Método   | Descripción                                                                                                                                                                                                                                                                                                                                                                                                              | Cuerpo de la Solicitud                           |
| ------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------ |
| `numerator`              | `GET`    | Recupera el valor actual del numerator, independientemente del estado del bloqueo. Siempre retorna el valor actual, incluso si el repositorio está bloqueado.                                                                                                                                                                                                                                                            | -                                                |
| `numerator`              | `PUT`    | Establece el valor del numerator al `value` especificado inmediatamente, sin verificar si el repositorio está bloqueado.                                                                                                                                                                                                                                                                                                 | `{ "value": <number> }`                          |
| `numerator/test-and-set` | `PUT`    | Establece atómicamente el numerator a `newValue` si el valor actual coincide con `oldValue`. Retorna `newValue` en caso de éxito, o error HTTP 400 con un cuerpo `{ "error": "Numerator does not match the expected old value.", "currentNumerator": <number> }` si falla. Esta operación es atómica, asegurando que la comparación y establecimiento del nuevo valor no puedan ser interrumpidas por otras operaciones. | `{ "oldValue": <number>, "newValue": <number> }` |
| `numerator/lock`         | `POST`   | Establece la bandera de bloqueo (`lock = true`) en el repositorio numerator. Hay un parámetro de timeout que es la cantidad de tiempo que el sistema seguirá intentando adquirir el bloqueo (por defecto es 10.000 milisegundos o 10 segundos). Retorna 400 si no se obtiene el lock antes de alcanzar el timeout. Solo una solicitud puede mantener el bloqueo a la vez, y el bloqueo NO se libera automáticamente.     | `{ "timeout": <number, in milliseconds> }`       |
| `numerator/lock`         | `DELETE` | Libera el bloqueo estableciendo la bandera de bloqueo a `false`. Si ya está `false`, permanece sin cambios.                                                                                                                                                                                                                                                                                                              | -                                                |

## Comprobaciones E2E y ejemplos

Ejemplos rápidos para validar localmente (asumiendo que el stack está levantado con `docker compose up`):

- Health:

```bash
curl -sS http://localhost:4000/health | jq
```

- Crear una transacción (debit):

```bash
curl -sS -X POST http://localhost:4000/transactions \
  -H "Content-Type: application/json" \
  -d '{
    "value":"100",
    "description":"T-Shirt Black M",
    "method":"debit_card",
    "cardNumber":"1234 5678 9012 3456",
    "cardHolderName":"Juan Pérez",
    "cardExpirationDate":"04/28",
    "cardCvv":"123"
  }'
```

- Obtener / listar / borrar:

```bash
curl -sS http://localhost:4000/transactions | jq
curl -sS http://localhost:4000/transactions/{id} | jq
curl -i -X DELETE http://localhost:4000/transactions/{id}

curl -sS http://localhost:4000/receivables | jq
curl -sS http://localhost:4000/receivables/{id} | jq
curl -i -X DELETE http://localhost:4000/receivables/{id}
```

Nota sobre CVV: por razones de seguridad, el CVV no se persiste ni se expone por la API. El campo se valida al recibir la petición pero no se serializa en las respuestas ni en la base de datos.

La documentación Swagger está disponible en `/docs` y el spec OpenAPI en `/openapi.yaml`.

Las pruebas end-to-end se ejecutan con Testcontainers en Go desde el paquete `internal/e2e`.
