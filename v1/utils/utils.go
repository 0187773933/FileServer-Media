package utils

import (
	"fmt"
	// "time"
	// "runtime"
	filepath "path/filepath"
	"strings"
	"strconv"
	"io/ioutil"
	"net/url"
	"context"
	"gopkg.in/yaml.v3"
	redis "github.com/redis/go-redis/v9"
	slug "github.com/gosimple/slug"
	types "github.com/0187773933/FileServer-Media/v1/types"
	// fiber_cookie "github.com/gofiber/fiber/v2/middleware/encryptcookie"
	// encryption "github.com/0187773933/encryption/v1/encryption"
	circular_set "github.com/0187773933/RedisCircular/v1/set"
	server "github.com/0187773933/GO_SERVER/v1/server"
)

func IsURL( input string ) ( result bool ) {
	result = false
	parsed_url , err := url.Parse( input )
	fmt.Println( parsed_url , parsed_url.Scheme )
	if err == nil {
		if parsed_url.Scheme == "http" || parsed_url.Scheme == "https" {
			result = true
		}
	}
	return
}

type Entry struct {
	Name string `yaml:"-"`
	Extension string  `yaml:"extension,omitempty"`
	ID string `yaml:"id,omitempty"`
	Length int `yaml:"length,omitempty"`
	Path string `yaml:"path,omitempty"`
	Items []*Entry `yaml:"items,omitempty"`
}

func ( e *Entry ) UnmarshalYAML( value *yaml.Node ) error {
	if value.Kind == yaml.MappingNode {
		for i := 0; i < len( value.Content ); i += 2 {
			keyNode := value.Content[ i ]
			valueNode := value.Content[ ( i + 1 ) ]
			if keyNode.Value == "extension" {
				e.Extension = valueNode.Value
			} else if keyNode.Value == "id" {
				e.ID = valueNode.Value
			} else if keyNode.Value == "length" {
				fmt.Sscanf( valueNode.Value , "%d" , &e.Length )
			} else if keyNode.Value == "path" {
				e.Path = valueNode.Value
			} else {
				e.Name = keyNode.Value
				var items []*Entry
				if err := valueNode.Decode( &items ); err != nil {
					return err
				}
				e.Items = items
			}
		}
	} else if value.Kind == yaml.SequenceNode {
		for _ , itemNode := range value.Content {
			var item Entry
			if err := itemNode.Decode( &item ); err != nil {
				return err
			}
			e.Items = append( e.Items , &item )
		}
	}
	return nil
}

func FlattenEntries( entries []*Entry , flattened *[]Entry ) {
	for _, entry := range entries {
		*flattened = append( *flattened , *entry )
		if entry.Items != nil {
			FlattenEntries( entry.Items , flattened )
		}
	}
}

func GetFlattenedEntries( yaml_file_path string ) ( result []Entry ) {
	fmt.Println( "Reading YAML file:" , yaml_file_path )
	data , err := ioutil.ReadFile( yaml_file_path )
	if err != nil { fmt.Printf( "Error reading YAML file: %v\n" , err ); return }
	var entries []*Entry
	err = yaml.Unmarshal( data , &entries );
	if err != nil { fmt.Printf( "Failed to Unmarshal YAML: %v\n" , err ); return }
	FlattenEntries( entries , &result )
	return
}

func RedisGetBool( ctx context.Context , client *redis.Client , key string ) ( result bool ) {
	result_str := client.Get( ctx , key ).Val()
	result , _ = strconv.ParseBool( result_str )
	return
}

func DeleteKeysWithPattern( ctx context.Context , db *redis.Client , pattern string ) error {
	var cursor uint64
	var keys []string
	var err error
	for {
		keys, cursor, err = db.Scan( ctx , cursor , pattern , 0 ).Result()
		if err != nil { return err }
		if len( keys ) > 0 {
			if _ , err := db.Del( ctx , keys... ).Result(); err != nil {
				return err
			}
		}
		if cursor == 0 {
			break
		}
	}
	return nil
}

func ImportLibrarySaveFilesInToRedis( s *server.Server ) {
	var ctx = context.Background()
	// defer redis_connection.FlushDB( ctx ).Err()
	files , _ := ioutil.ReadDir( s.Config.SaveFilesPath )
	for _ , file := range files {

		if file.IsDir() { continue }
		if strings.HasSuffix( file.Name() , ".yaml" ) == false { continue }
		file_path := filepath.Join( s.Config.SaveFilesPath , file.Name() )
		library_entries := GetFlattenedEntries( file_path )
		total_entries := len( library_entries )
		if total_entries < 1 { continue }

		// global_circular_key := fmt.Sprintf( "%s.%s", s.Config.Redis.Prefix , library_entry.RedisKey )
		file_name := strings.TrimSuffix( file.Name() , filepath.Ext( file.Name() ) )
		file_name_slug := slug.Make( file_name )
		global_circular_key := fmt.Sprintf( "%s.%s" , s.Config.Redis.Prefix , file_name_slug )

		s.REDIS.SAdd( ctx , s.Config.Redis.Prefix + ".LIBRARY" , file_name_slug )

		// Force Reset
		redis_reset := true
		if redis_reset == true {
			// fmt.Println( "resetting global circular key:" , global_circular_key , file_path , total_entries )
			s.REDIS.Del( ctx , global_circular_key )
			// db.Del( ctx, global_circular_key + ".INDEX" )
		}

		for index, entry := range library_entries {
			if entry.Path == "" {
				continue
			}
			if entry.ID == "" {
				continue
			}
			fmt.Printf( "index: %d, name: %s, path: %s, id: %s\n" , index , entry.Name , entry.Path , entry.ID )
			global_entry_key := fmt.Sprintf( "%s.%s" , s.Config.Redis.Prefix , entry.ID )

			// Minimum need the path, could json blob store here instead
			s.REDIS.Set( ctx , global_entry_key , entry.Path , 0 )
			// So these are setting up "sessions"
			// a "session" here is just an ephemeral copy of the circular set's index tracking
			fmt.Println( "adding" , global_circular_key , entry.ID )
			circular_set.Add( s.REDIS , global_circular_key , entry.ID )
		}
		fmt.Println( "done" )
	}
}

func GetMediaHTML( params types.GetMediaHTMLParams ) ( html string ) {
	if params.Extension == "mp4" || params.Extension == "webm" {
		html = fmt.Sprintf(`
			<!DOCTYPE html>
			<html lang="en">
			<head>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<title>Video Player</title>
				<style>
					body, html {
						margin: 0;
						padding: 0;
						height: 100%%;
						overflow: hidden;
						display: flex;
						justify-content: center;
						align-items: center;
						background: black;
					}
					video {
						width: 100%%;
						height: 100%%;
						object-fit: contain;
					}
					.overlay {
						position: absolute;
						top: 0;
						left: 0;
						width: 100%%;
						height: 100%%;
						display: flex;
						justify-content: center;
						align-items: center;
						background: rgba(0, 0, 0, 0.5);
						color: white;
						font-size: 24px;
						cursor: pointer;
					}
				</style>
			</head>
			<body>
				<video id="mediaPlayer" controls>
					Your browser does not support the video tag.
				</video>
				<div id="overlay" class="overlay">Click to Play</div>
				<script>
					document.addEventListener("DOMContentLoaded", () => {
						const media = document.getElementById('mediaPlayer');
						const overlay = document.getElementById('overlay');
						const session_key = "%s";
						const files_prefix = "%s";
						const library_key = "%s";
						const session_id = "%s";
						const time_str = "%s";
						const uuid = "%s";
						const extension = "%s";
						const ready_url = "%s";
						const media_type = "%s";
						console.log( session_key , files_prefix , library_key , session_id , time_str , uuid , extension );
						const media_src = "/" + files_prefix + "/" + uuid + "." + extension;
						console.log( media_src );
						media.src = media_src;
						let signaled_ready_fresh = false;
						let signaled_fresh = false;
						media.addEventListener( 'loadedmetadata' , () => {
							if ( !signaled_ready_fresh ) {
								signaled_ready_fresh = true;
								try {
									let ready_fresh_url = ready_url.replace( "ready" , "readyfresh" );
									console.log( ready_fresh_url );
									fetch( ready_fresh_url , { method: 'GET' });
								} catch( e ) { console.log( e ); }
							}
							try {
								if ( time_str !== "") {
									console.log('Setting time to', parseInt(time_str));
									let x_time = parseInt( time_str );
									if ( x_time > 2 ) {
										let offset = ( x_time - 1 );
										console.log( "using offset" , offset );
										media.currentTime = offset;
									}
								}
							} catch( e ) { console.log( e ); }
						});

						overlay.addEventListener( 'click' , async () => {
							overlay.style.display = 'none';
							try {
								media.play().then(() => {
									try {
										if (media.requestFullscreen) {
											media.requestFullscreen();
										} else if (media.mozRequestFullScreen) {
											media.mozRequestFullScreen();
										} else if (media.webkitRequestFullscreen) {
											media.webkitRequestFullscreen();
										} else if (media.msRequestFullscreen) {
											media.msRequestFullscreen();
										}
										if (time_str !== "") {
											console.log('Setting time to', parseInt(time_str));
											let x_time = parseInt( time_str );
											if ( x_time > 2 ) {
												let offset = ( x_time - 1 );
												console.log( "using offset" , offset );
												media.currentTime = offset;
											}
										}
									} catch ( e ) { console.log( e ); }
									if ( !signaled_fresh ) {
										signaled_fresh = true;
										try {
											console.log( ready_url );
											fetch( ready_url , { method: 'GET' });
										}
										catch( e ) { console.log( e ); }
									}
								}).catch(error => {
									console.error('Error attempting to play media:', error);
								});
							} catch( e ) { console.log( e ); }
						});

						let last_time_update = 0;
						media.addEventListener( 'timeupdate' , () => {
							let x_time = Math.round( media.currentTime );
							if ( x_time === last_time_update ) { return; }
							last_time_update = x_time;
							let duration = Math.round( media.duration );
							let finished = false;
							if ( x_time >= ( duration - 1 ) ) { finished = true; }
							console.log( x_time , duration , finished );
							fetch( '/update_position' , {
								method: 'POST',
								headers: { 'Content-Type': 'application/json' , "k": session_key } ,
								body: JSON.stringify({ library_key: library_key , session_id: session_id , uuid: uuid , position: last_time_update , duration: duration , finished: finished })
							});
							if (finished) {
								setTimeout(() => {
									document.exitFullscreen().then(() => {
										let url = new URL(window.location.href);
										url.searchParams.set( 'ready_url' , ready_url );
										window.location.href = url.toString();
									}).catch((err) => {
										console.error('Error attempting to exit full-screen mode: ', err);
										let url = new URL(window.location.href);
										url.searchParams.set( 'ready_url' , ready_url );
										window.location.href = url.toString(); // Fallback to refresh even if exit fullscreen fails
									});
								}, 1000);
							}
						});
					});
				</script>
			</body>
			</html>
		`, params.SessionKey , params.FilesURLPrefix , params.LibraryKey , params.SessionID , params.TimeStr , params.NextID , params.Extension , params.ReadyURL , params.Type )
	} else if ( params.Extension == "mp3" || params.Extension == "wav" || params.Extension == "ogg") {
		html = fmt.Sprintf(`
			<!DOCTYPE html>
			<html lang="en">
			<head>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<title>Audio Player</title>
				<style>
					body, html {
						margin: 0;
						padding: 0;
						height: 100%%;
						overflow: hidden;
						display: flex;
						justify-content: center;
						align-items: center;
						background: black;
						color: white;
						font-size: 24px;
					}
					audio {
						width: 100%%;
					}
					.overlay {
						position: absolute;
						top: 0;
						left: 0;
						width: 100%%;
						height: 100%%;
						display: flex;
						justify-content: center;
						align-items: center;
						background: rgba(0, 0, 0, 0.5);
						color: white;
						font-size: 24px;
						cursor: pointer;
					}
				</style>
			</head>
			<body>
				<audio id="mediaPlayer" controls>
					Your browser does not support the audio tag.
				</audio>
				<div id="overlay" class="overlay">Click to Play</div>
				<script>
					document.addEventListener("DOMContentLoaded", () => {
						const media = document.getElementById('mediaPlayer');
						const overlay = document.getElementById('overlay');
						const session_key = "%s";
						const files_prefix = "%s";
						const library_key = "%s";
						const session_id = "%s";
						const time_str = "%s";
						const uuid = "%s";
						const extension = "%s";
						const ready_url = "%s";
						const media_type = "%s";
						console.log( session_key , files_prefix , library_key , session_id , time_str , uuid , extension );
						const media_src = "/" + files_prefix + "/" + uuid + "." + extension;
						console.log( media_src );
						media.src = media_src;
						let signaled_ready_fresh = false;
						let signaled_fresh = false;
						let update_count = 0;
						media.addEventListener( 'loadedmetadata' , () => {
							if ( !signaled_ready_fresh ) {
								signaled_ready_fresh = true;
								try {
									let ready_fresh_url = ready_url.replace( "ready" , "readyfresh" );
									console.log( ready_fresh_url );
									fetch( ready_fresh_url , { method: 'GET' });
								} catch( e ) { console.log( e ); }
							}
							try {
								if ( time_str !== "") {
									console.log('Setting time to', parseInt(time_str));
									let x_time = parseInt( time_str );
									if ( x_time > 2 ) {
										let offset = ( x_time - 1 );
										console.log( "using offset" , offset );
										media.currentTime = offset;
									}
								}
							} catch( e ) { console.log( e ); }
						});

						overlay.addEventListener( 'click' , async () => {
							overlay.style.display = 'none';
							try {
								media.play().then(() => {
									if ( !signaled_fresh ) {
										signaled_fresh = true;
										try {
											console.log( ready_url );
											fetch( ready_url , { method: 'GET' });
										}
										catch( e ) { console.log( e ); }
									}
								}).catch(error => {
									console.error('Error attempting to play media:', error);
								});
							} catch( e ) { console.log( e ); }
						});

						let last_time_update = 0;
						media.addEventListener( 'timeupdate' , () => {
							let x_time = Math.round( media.currentTime );
							if ( x_time === last_time_update ) { return; }
							last_time_update = x_time;
							let duration = Math.round( media.duration );
							let finished = false;
							if ( x_time >= ( duration - 1 ) ) { finished = true; }
							console.log( x_time , duration , finished );
							fetch( '/update_position' , {
								method: 'POST',
								headers: { 'Content-Type': 'application/json' , "k": session_key } ,
								body: JSON.stringify({ library_key: library_key , session_id: session_id , uuid: uuid , position: last_time_update , duration: duration , finished: finished })
							});
							update_count += 1;
							if ( update_count >= 3 ) {
								if (finished) {
									setTimeout(() => {
										console.log( "calling refresh ???" );
										let url = new URL(window.location.href);
										url.searchParams.set( 'ready_url' , ready_url );
										window.location.href = url.toString();
									}, 1000);
								}
							}
						});
					});
				</script>
			</body>
			</html>
		`, params.SessionKey , params.FilesURLPrefix , params.LibraryKey , params.SessionID , params.TimeStr , params.NextID , params.Extension , params.ReadyURL , params.Type )
	}
	return
}

func GetYouTubePlaylistHTML( params types.GetYouTubePlaylistParams ) ( html string ) {
	html = fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>YouTube Playlist</title>
	<style>
		body, html {
			height: 100%%;
			width: 100%%;
			margin: 0;
			display: flex;
			justify-content: center;
			align-items: center;
			background-color: black;
		}
		#yt-wrap {
			width: 100%%;
			height: 100%%;
			display: flex;
			justify-content: center;
			align-items: center;
		}
		#ytplayer {
			width: 100%%;
			height: 100%%;
		}
		#playButton {
			position: absolute;
			top: 50%%;
			left: 50%%;
			transform: translate(-50%%, -50%%);
			background-color: white;
			padding: 10px 20px;
			cursor: pointer;
			z-index: 1;
		}
	</style>
</head>
<body>
	<div id="yt-wrap">
		<div id="ytplayer"></div>
		<div id="playButton">Play</div>
	</div>
	<script>
		const session_key = "%s";
		const library_key = "%s";
		const playlistId = "%s";
		const session_id = "%s";
		const startTime = "%s";
		const startIndex = "%s";
		const ready_url = "%s";
		var tag = document.createElement('script');
		tag.src = "https://www.youtube.com/player_api";
		var firstScriptTag = document.getElementsByTagName('script')[0];
		firstScriptTag.parentNode.insertBefore(tag, firstScriptTag);
		function onYouTubePlayerAPIReady() {
			let x = new YT.Player('ytplayer', {
				width: '100%%',
				height: '100%%',
				playerVars: {
					'autoplay': 0,
					'playsinline': 1,
				},
				events: {
					'onReady': onPlayerReady,
					'onStateChange': onPlayerStateChange
				}
			});
			window.player = x;
			window.LAST_UPDATED_TIME = 0;
			startTrackingPosition();
		}
		function onPlayerReady(event) {
			document.getElementById('playButton').addEventListener('click', () => {
				window.player.loadPlaylist({
					list: playlistId,
					index: parseInt( startIndex ) ,
					startSeconds: parseInt( startTime ) ,
				});
				window.player.playVideo();
				document.getElementById('playButton').style.display = 'none';
			});
		}
		function setMaxQuality() {
			const qualities = window.player.getAvailableQualityLevels();
			if (qualities.length) {
				console.log( "setting quality to" , qualities[0] );
				window.player.setPlaybackQuality(qualities[0]);
			}
		}
		function onPlayerStateChange(event) {
			if (event.data === YT.PlayerState.ENDED) {
				if (window.player.getPlaylistIndex() < window.player.getPlaylist().length - 1) {
					window.player.nextVideo();
				}
			} else if (event.data === YT.PlayerState.PLAYING) {
				setMaxQuality();
			}
		}
		function getCurrentVideoId() {
			const videoUrl = window.player.getVideoUrl();
			const urlParams = new URLSearchParams(new URL(videoUrl).search);
			const videoId = urlParams.get('v');
			return videoId;
		}
		function getCurrentVideoTitle() {
			const videoData = window.player.getVideoData();
			const videoTitle = videoData.title;
			return videoTitle;
		}
		function startTrackingPosition() {
			setInterval(() => {
				if ( !window.player ) { return; }
				const currentTime = parseInt( window.player.getCurrentTime() );
				if ( currentTime === window.LAST_UPDATED_TIME ) { return; }
				window.LAST_UPDATED_TIME = currentTime;
				const duration = parseInt( window.player.getDuration() );
				const videoId = getCurrentVideoId();
				const videoTitle = getCurrentVideoTitle();
				const playlistIndex = window.player.getPlaylistIndex();
				let info = {
					library_key: playlistId ,
					session_id: session_id ,
					library_key: library_key ,
					youtube_playlist_id: playlistId ,
					youtube_playlist_index: playlistIndex ,
					title: videoTitle ,
					position: currentTime ,
					duration: duration ,
					ready_url: ready_url ,
					type: "youtube-playlist" ,
				};
				console.log( info );
				fetch( '/update_position' , {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' , "k": session_key } ,
					body: JSON.stringify( info )
				});
			}, 1000 );
		}
	</script>
</body>
</html>` , params.SessionKey , params.LibraryKey , params.PlaylistID , params.SessionID , params.Time , params.Index , params.ReadyURL )
	return
}