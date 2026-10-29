# Interesse Popular — API interna de classificação

Esta é uma API de administração isolada. Ela concentra a classificação editorial de partidos, as referências usadas e as versões da metodologia. Não possui rotas públicas para leitura ou alteração desses dados.

## Executar localmente

1. Copie `.env.example` para `.env` e defina uma chave segura em `INTERNAL_API_KEY`.
2. Inicie o PostgreSQL com `docker compose up -d postgres`.
3. Execute `go run .`.

Com `AUTO_MIGRATE=true`, a tabela `party_political_directions` é criada ou atualizada no início da aplicação.

## Rotas

`GET /api/v1/healthcheck` é a única rota sem chave. O CRUD está sob `/api/internal/v1/party-political-directions` e exige o cabeçalho `X-Internal-API-Key`:

```text
POST   /api/internal/v1/party-political-directions
GET    /api/internal/v1/party-political-directions
GET    /api/internal/v1/party-political-directions/:id
PUT    /api/internal/v1/party-political-directions
DELETE /api/internal/v1/party-political-directions/:id
```

`INTERNAL_API_KEY` pode acessar todas as rotas. `INTERNAL_READ_API_KEY` permite
somente os `GET`s. O filtro
`GET /api/internal/v1/party-political-directions?direction=direita` retorna apenas
classificações ativas para aquela direção.

`POST` e `PUT` recebem um array. A criação e a edição são processadas item a item;
uma falha não cancela os outros itens. A resposta é `207 Multi-Status` e mantém a
posição original em `index`. Sigla, nome e número do TSE são únicos.

Exemplo de corpo para criação:

```json
[
  {
    "partyTseNumber": 99,
    "partyAcronym": "SIGLA",
    "partyName": "Nome do partido",
    "direction": "esquerda",
    "ideology": "esquerda_liberal",
    "methodologyVersion": "v1.0",
    "sourceName": "Referência consultada",
    "sourceUrl": "https://exemplo.org/fonte",
    "evidenceNote": "Justificativa editorial e evidências verificáveis.",
    "validFrom": "2026-10-28T00:00:00Z"
  }
]
```

Para editar, use o mesmo objeto incluindo `id`. Em caso de conflito, por exemplo
uma sigla já existente, o item retorna `status: "error"` e `error.code: "conflict"`,
enquanto os demais podem retornar `status: "success"`.

## Formato de respostas

Listas usam `{"items": [], "total": 0}` e confirmações simples retornam
`{"message":"success"}`. Erros usam o formato do `go-commons`:
`{"status": 400, "messages": ["..."]}`. Os resultados de operações em lote
ficam dentro de `items`.

Valores aceitos:

- `direction`: `esquerda` ou `direita`.
- `ideology`: `esquerda_radical`, `esquerda_liberal`, `direita_liberal` ou `extrema_direita`.

O serviço valida a coerência entre direção e ideologia. `methodologyVersion`, fonte, período de vigência e nota de evidência tornam cada classificação auditável. Nenhum partido é inserido automaticamente: classificações são decisões editoriais registradas pela equipe autorizada.

## Cache de parlamentares

Esta API consulta Câmara e Senado, associa a classificação ativa de cada partido aos
parlamentares e grava o resultado no Redis. A atualização periódica é de uma hora,
com verificação de cache vazio a cada minuto e TTL de oito horas. Criações, edições e
exclusões de classificações solicitam uma atualização imediatamente; se outra já
estiver em execução, uma nova é enfileirada ao seu término.

## Políticos, mandatos e cartas

O sincronizador mantém todo o conjunto de políticos, não apenas quem está em
exercício: a identidade interna de cada pessoa, seus IDs externos e seus mandatos.
O ID da Câmara ou do Senado identifica a origem do dado; o `politician.id` é a
identidade permanente usada pela plataforma. A fonte eleitoral do TSE é configurada
por `TSE_ELECTED_CANDIDATES_URL` e deve apontar para o ZIP/CSV oficial com a coluna
`DS_SIT_TOT_TURNO` (o exemplo usa `consulta_cand_2026.zip`).

O mesmo político pode ter, simultaneamente, o mandato atual `active` e o próximo
`elected_pending`. Este último contém início e fim previstos. Quando a casa passa a
listá-lo após a posse, o próximo mandato vira `active` e o anterior vira `former`.
Quando alguém deixa de constar na lista oficial, seu mandato também é preservado como
`former`; a última observação da fonte e a data de encerramento ficam registradas.
Deputados federais em exercício recebem as datas constitucionais inferidas do mandato
vigente; para senadores em exercício, a data só é preenchida quando a fonte informar
o mandato ou quando ele vier do resultado eleitoral, pois as turmas têm vencimentos
diferentes.

O sync só faz correspondência automática quando nome normalizado, UF e cargo formam
uma única correspondência. A associação é registrada para auditoria. Não há merge
automático por nome isolado.

As cartas são uma camada editorial manual e não são inferidas pelo partido ou pelo
sync. Primeiro consulte `GET /api/internal/v1/politicians` para obter o
`politicianId`; depois use o CRUD protegido:

```text
GET    /api/internal/v1/politician-editorial-marks
POST   /api/internal/v1/politician-editorial-marks
PUT    /api/internal/v1/politician-editorial-marks
DELETE /api/internal/v1/politician-editorial-marks/:id
```

`POST` e `PUT` recebem arrays e retornam `207 Multi-Status`, como a classificação de
partidos. Cada marca exige direção, ideologia, evidência, fonte, versão de metodologia
e período de vigência; `cardEnabled` controla se ela pode ser exibida como carta.
