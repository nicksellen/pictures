# pictures

The beginnings of a tool to help me manage my pictures.

First task for it is to create a local directory of thumbnails from my various sources:
- local hard disk
- USB connected hard disk
- backblaze b2 cloud storage

Other things I might want it to do:
- have a web frontend that can browse the images/thumbnails, filtering by rating/tag, etc
- have a way to shared a filtered subset of them with people online
- let authorized people download collections
- allow to associate _rendered_ versions of the images (e.g. fiddle with in shotwell, export with crop/filters/etc)
- somehow integrate with the image processing metadata (crop, colour mods, etc)
- manage the upload/download/sync of the photos (currently I use https://github.com/nicksellen/imageimport) to put in structured directories then rclone to sync to b2

```
# inside your gopath src...
git clone git@github.com:nicksellen/pictures.git
cd pictures
go build
export B2_ACCOUNT_ID=youraccountid B2_ACCOUNT_KEY=youraccountkey
./pictures b2://bucketname/prefix
```
