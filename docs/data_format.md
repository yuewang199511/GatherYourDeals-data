# Purchase Data Format

As each user will need to record their own purchase data, the user is responsible for performing the ETL process.

Although the ETL process can be helped by other applications/framework in the future, we need to have the standard for it as reference.

# ETL Structure

## Raw Data
In order to get the best deals, we usually go online and try to memorize where the best deals are and we check on the receipts to make sure we have the deals applied. So for GatherYourDeals, the raw resource of data is usually from:

1. online search
2. Physical/digital receipts owned by the maintainer

⚠️⚠️⚠️ In order to stay on legal ground, we do not encourage automatically scrape data from merchants!

## Extracted Data

The thing we care about most about each receipt, is where, when, what, and how much we bought an item. This information can be easily expressed as json.

However, someone may want to record the background and the amount of deals, so they have better understanding of whether the prices is stable or not.

We will not be able to design a profound static json format to store all possible fields, so we will also use a meta.json for the user to describe any extra fields they provide.

## Load Data

The solution we provide is mainly for private, small group usage, so the user should perform the ETL process and the lifecycle of the data. Thus there will be interleavings between the storage of the extracted data and the loaded data for production.

In order to address this, we will separate these two databases and provide a solution for loading.

# Data Format

The record format will follow the **First Normal Form** to have a clean starting point for future querying.

## Receipt Data

The receipt data should be **provided** as a list of dictionaries in this format:

⚠️⚠️⚠️ At this stage, data might only exist as files scattered around, so we define it as **provided** rather than **recorded**

````json

{
    "productName": "name of the item",
    "purchaseDate": 2025.04.05,
    "price": "1.56CAD",
    "amount": "1 or 2lb",
    "storeName": "some store",
    "latitude": 49.2827,
    "longitude": -123.1207,
    "extrakeys": ......
}

````

## Metadata

In order to provide a flexible database that can accommodate possible analysis and user features, we allow the data maintainer to add any possible fields and explain their usage.

💡💡💡 We assume that by providing a description of the fields, it is enough for LLM models to adapt to user requirements to some extent, and later for human developers to build features upon certain fields.


In order to record your data with customized fields, you need the **staging dataset** to read a list of dictionaries in this format:

````json

{
    "fieldName": "name of the field",
    "description": "description of the field"
}

````

### Data Verification with Metadata

When a new record is inserted, only when all fields exist in the ``meta`` table in the **staging database** that it can be inserted, else it will be rejected.

### Update the Fields

When a new meta is uploaded, if a field already exists, user will not be notified which fields already exist and the uploading process will fail.

The client will provide a update meta API to update the description of a certain field.

💡💡💡 The intuition behind this process is to force user to observe the inconsistency in their data definitions.

## Native Keys Definition

These keys will always exist to support the definitions of fields that the user cannot change:

| fieldName    | description    | type          |
|:-------------|:--------------:|--------------:|
| productName  | name of product| string        |
| purchaseDate | purchase date in Y.M.D| string |
| price | the price for payment | string |
| amount | the amount of purchased goods, in the format of ``number`` or ``number(unit)`` | string |
| storeName | Name of the store | string |
| latitude | latitude of the location, this field is optional | float |
| longitude | longitude  of the location, this field is optional | float |

💡💡💡 the type here is just for general type definition, not referring to any specific language or database

## Tracking of Records

In the early stage of this project, we will not go to the extent of event sourcing to ensure every data record can be **recovered** even if the original extracted jsons are lost. We only provide means to **track** the resource of the records.

For each record, there will be these fields appended


````json

{
    "uploadTime": 1770620311,
    "userID": "registered user ID   "

}

````


| fieldName    | description    | type          |
|:-------------|:--------------:|--------------:|
| uploadTime  | upload time in epoch timestamp in seconds| int |
| userId | the id of the user who operated, this is just for possible team features and tracking.| string |