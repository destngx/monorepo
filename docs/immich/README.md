# Immich photo server plan

Status: planned, not set up yet. Nothing in this repo deploys it.

## Goal

Reach every photo and video from any device, back up phones and tablets automatically, keep originals at full quality, and pay nothing per month.

Sources:

- iPhone and iPad, through the Immich iOS app.
- A camera, through its SD card.

## Decisions

| Topic           | Decision                                                            |
| --------------- | ------------------------------------------------------------------- |
| Photo app       | [Immich](https://immich.app), self-hosted                           |
| Server          | An always-on Windows PC, Immich in Docker on WSL2                   |
| Remote access   | [Tailscale](https://tailscale.com), no open router ports            |
| Library storage | One HDD for originals, the PC's SSD for the database and thumbnails |
| Backup          | Nightly lossless restic snapshots to a second HDD                   |
| Offsite copy    | Not yet. Add a rotated third HDD or Backblaze B2 later              |
| Compression     | Lossless only. Same quality as the originals                        |

## Alternatives considered

- **Cloudinary.** The free tier is small and video bandwidth costs money. See the [archived app](../cloudinary-photos-app/README.md).
- **Google Photos, iCloud, OneDrive.** No setup, but the free tiers (5 to 15 GB) are far below the library size.
- **Ente.** End-to-end encrypted and open source, but the free tier is small.
- **Synology DS223j.** Its ARM CPU and 1 GB RAM cannot run Immich, and DSM 7.2.2 removed server-side HEVC processing, which iPhone video needs.
- **Synology DS224+ with two mirrored drives.** The best long-term option: owned hardware, a mirror, low power use, and it runs Immich in Container Manager. It costs about four times the HDD-only plan. Moving to it later means copying the library folder and restoring a database dump.

## Storage layout

| Drive  | Path        | Holds                                                                              |
| ------ | ----------- | ---------------------------------------------------------------------------------- |
| PC SSD | WSL2        | Docker, the Postgres volume, thumbnails (optional)                                 |
| HDD 1  | `D:\immich` | Immich upload location: `library`, `upload`, `profile`, `backups`, `encoded-video` |
| HDD 2  | `E:\restic` | Local restic repository                                                            |

Why the split:

- Originals are written once and read now and then, which an HDD handles well and cheaply.
- The timeline reads thousands of thumbnails and runs many small Postgres queries. On an HDD the grid fills in slowly. Immich's docs recommend an SSD for the database.
- On Windows, Postgres must live in a Docker volume inside WSL, not in a Windows folder such as `D:\`. A Windows folder causes permission errors and corruption.

Drive choice:

- 3.5" desktop drives with their own power adapter suit a server that runs all day better than 2.5" bus-powered portables, which are usually SMR and slow down during large imports.
- Size each HDD at several times the current library. Video grows fast, and Immich stores transcoded copies next to the originals.
- Turn off the USB enclosure's sleep timer so the drive does not disconnect.

## Server setup

1. Install WSL2 with Ubuntu.
2. Install Docker. Docker Desktop is free only for personal use and companies under 250 employees and $10M revenue. Otherwise install Docker Engine inside WSL2 Ubuntu.
3. Run Immich from its official `docker-compose.yml`, with `UPLOAD_LOCATION` on HDD 1 and `DB_DATA_LOCATION` as a Docker volume.
4. In Immich admin settings, turn on the automatic daily database dump. It writes to `UPLOAD_LOCATION/backups`.
5. Install Tailscale on the PC and on every device.
6. Keep the PC available: turn off sleep, set Windows Update active hours, and start WSL and Docker at boot from Task Scheduler.

## Getting files in

- **iPhone and iPad.** The Immich app backs up the camera roll over Tailscale. iOS limits background work, so open the app now and then.
- **Camera.** Copy the SD card to a folder on HDD 1 and add it to Immich as an External Library. This keeps raw files and full-quality video. Sending files through the phone app works too, but limits raw files and long videos.
- **Camera setting.** Shoot "Lossless Compressed RAW" where the camera offers it. Raw files get much smaller with no quality loss.
- **Existing library.** Use `immich-go` for Google Takeout or iCloud exports, and the Immich CLI for plain folders.

## Backup

### Why no live mirror

Windows Storage Spaces can mirror two drives, but a mirror copies deletions, corrupted files, and ransomware to both drives at once, and a USB drive dropping off degrades the pool. Nightly snapshots on a second drive cover a dead drive and also restore yesterday's version of a file. The worst case is losing one day of new files, and the rule in [Freeing device space](#freeing-device-space) covers that day.

### Why lossless only

HEIC, HEVC, JPEG, compressed ARW, and XAVC are already compressed, so restic's zstd saves about 0 to 5%. A smaller copy would need re-encoding video, which loses quality. Plan for the backup to be about the size of the library.

### What restic backs up

Back up `library`, `upload`, `profile`, and `backups` from `D:\immich`. Skip `thumbs` and `encoded-video`, which Immich regenerates.

### Schedule (Windows Task Scheduler)

| When        | Job                                                                                 |
| ----------- | ----------------------------------------------------------------------------------- |
| 02:00 daily | Immich database dump (built in)                                                     |
| 03:00 daily | `restic backup --compression max` from `D:\immich` to `E:\restic`                   |
| Sunday      | `restic forget --prune --keep-daily 7 --keep-weekly 4 --keep-monthly 12`            |
| Monthly     | `restic check --read-data-subset=5%`                                                |
| Quarterly   | Restore a few photos and the latest database dump to a scratch folder and open them |

Keep the restic repository password in a password manager. Without it the backup cannot be decrypted.

### Freeing device space

Delete a file from its source only after the nightly backup includes it:

- iPhone and iPad: use "Free up space" in the Immich app after the next nightly run.
- Camera: format the SD card only after the nightly run.

### Failure cases

| Failure                                        | Recovery                                                                                     |
| ---------------------------------------------- | -------------------------------------------------------------------------------------------- |
| HDD 1 dies                                     | Restore from HDD 2                                                                           |
| A photo deleted by mistake                     | Immich trash (30 days), then a restic snapshot                                               |
| PC SSD dies                                    | Reinstall WSL and Docker, restore the database from the dump on HDD 1. Thumbnails regenerate |
| Fire, theft, surge, or losing access to the PC | Both drives are in the same place. Not covered until an offsite copy exists                  |

### Offsite copy (later)

Pick one:

- **Rotated drive.** Buy a third HDD, swap it with HDD 2 monthly, and keep the spare elsewhere. No monthly cost. The worst case is losing one month of new files.
- **Backblaze B2.** About $6 to $7 per TB per month, lossless and encrypted with restic. The PC uploads with a key that cannot delete, and pruning runs from another machine with a separate key, so ransomware on the PC cannot erase the cloud copy.

## Open questions

- Windows 10 or 11 on the server. Windows 11 makes WSL networking and auto-start easier.
- Docker Desktop or Docker Engine, which depends on the license terms above.
- Where the Compose file, `.env`, restic scripts, and Task Scheduler jobs will live.
