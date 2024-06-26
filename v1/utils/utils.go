package utils

import (
	"fmt"
	"time"
	"runtime"
	"strings"
	"strconv"
	"io/ioutil"
	"net/url"
	"context"
	"gopkg.in/yaml.v3"
	redis "github.com/redis/go-redis/v9"
	types "github.com/0187773933/FileServer-Media/v1/types"
	fiber_cookie "github.com/gofiber/fiber/v2/middleware/encryptcookie"
	encryption "github.com/0187773933/encryption/v1/encryption"
	// circular_set "github.com/0187773933/RedisCircular/v1/set"
)

func SetupStackTraceReport() {
	if r := recover(); r != nil {
		stacktrace := make( []byte , 1024 )
		runtime.Stack( stacktrace , true )
		fmt.Printf( "%s\n" , stacktrace )
	}
}

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

var location , _ = time.LoadLocation( "America/New_York" )
func GetFormattedTimeString() ( result string ) {
	time_object := time.Now().In( location )
	month_name := strings.ToUpper( time_object.Format( "Jan" ) )
	milliseconds := time_object.Format( ".000" )
	date_part := fmt.Sprintf( "%02d%s%d" , time_object.Day() , month_name , time_object.Year() )
	time_part := fmt.Sprintf( "%02d:%02d:%02d%s" , time_object.Hour() , time_object.Minute() , time_object.Second() , milliseconds )
	result = fmt.Sprintf( "%s === %s" , date_part , time_part )
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
			keyNode := value.Content[i]
			valueNode := value.Content[ i+1 ]
			if keyNode.Value == "extension" {
				e.Extension = valueNode.Value
			} else if keyNode.Value == "id" {
				e.ID = valueNode.Value
			} else if keyNode.Value == "length" {
				fmt.Sscanf(valueNode.Value, "%d", &e.Length)
			} else if keyNode.Value == "path" {
				e.Path = valueNode.Value
			} else {
				e.Name = keyNode.Value
				var items []*Entry
				if err := valueNode.Decode(&items); err != nil {
					return err
				}
				e.Items = items
			}
		}
	} else if value.Kind == yaml.SequenceNode {
		for _, itemNode := range value.Content {
			var item Entry
			if err := itemNode.Decode(&item); err != nil {
				return err
			}
			e.Items = append(e.Items, &item)
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

func GenerateNewKeys() {
	fiber_cookie_key := fiber_cookie.GenerateKey()
	encryption_key := encryption.GenerateRandomString( 32 )
	server_api_key := encryption.GenerateRandomString( 16 )
	admin_username := encryption.GenerateRandomString( 16 )
	admin_password := encryption.GenerateRandomString( 16 )
	session_key := encryption.GenerateRandomString( 16 )
	login_url := encryption.GenerateRandomString( 16 )
	files_url_prefix := encryption.GenerateRandomString( 8 )
	fmt.Println( "Generated New Keys :" )
	fmt.Printf( "\tFiber Cookie Key === %s\n" , fiber_cookie_key )
	fmt.Printf( "\tEncryption Key === %s\n" , encryption_key )
	fmt.Printf( "\tServer API Key === %s\n" , server_api_key )
	fmt.Printf( "\tAdmin Username === %s\n" , admin_username )
	fmt.Printf( "\tAdmin Password === %s\n" , admin_password )
	fmt.Printf( "\tSession Key === %s\n" , session_key )
	fmt.Printf( "\tLogin URL === %s\n" , login_url )
	fmt.Printf( "\tFiles URL Prefix === %s\n" , files_url_prefix )
	panic( "Exiting" )
}

func ParseConfig( file_path string ) ( result types.ConfigFile ) {
	config_file , _ := ioutil.ReadFile( file_path )
	error := yaml.Unmarshal( config_file , &result )
	if error != nil { panic( error ) }
	return
}

func RedisGetBool( ctx context.Context , client *redis.Client , key string ) ( result bool ) {
	result_str := client.Get( ctx , key ).Val()
	result , _ = strconv.ParseBool( result_str )
	return
}


func DeleteKeysWithPattern(ctx context.Context, db *redis.Client, pattern string) error {
	var cursor uint64
	var keys []string
	var err error

	for {
		keys, cursor, err = db.Scan(ctx, cursor, pattern, 0).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			if _, err := db.Del(ctx, keys...).Result(); err != nil {
				return err
			}
		}

		if cursor == 0 {
			break
		}
	}
	return nil
}


func GetMediaHTML(
	session_key string ,
	files_url_prefix string ,
	library_key string ,
	session_id string ,
	time_str string ,
	next_id string ,
	extension string ,
	ready_url string ,
) ( html string ) {
	if extension == "mp4" || extension == "webm" || extension == "ogg" {
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
		`, session_key, files_url_prefix, library_key, session_id, time_str, next_id, extension, ready_url)
	} else if (extension == "mp3" || extension == "wav" || extension == "ogg") {
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
		`, session_key, files_url_prefix, library_key, session_id, time_str, next_id, extension, ready_url)
	}
	return
}
