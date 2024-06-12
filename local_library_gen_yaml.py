import os
import sys
import uuid
import yaml
from pathlib import Path
import subprocess
from natsort import humansorted

SAVE_DURATION = False
SAVE_EXTENSION = False

common_video_extensions = {
	".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv", ".webm", ".mpg", ".mpeg",
	".m4v", ".3gp", ".3g2", ".f4v", ".f4p", ".f4a", ".f4b", ".ts", ".m2ts",
	".mxf", ".ogv", ".rm", ".rmvb", ".divx", ".vob", ".qt", ".yuv", ".asf",
	".mts", ".m2v", ".amv", ".svi"
}

common_audio_extensions = {
	".mp3", ".wav", ".aac", ".flac", ".alac", ".wma", ".ogg", ".m4a", ".aiff",
	".au", ".ra", ".rm", ".amr", ".ape", ".dsd", ".dts", ".mid", ".midi",
	".mpa", ".opus", ".voc", ".vox", ".mka", ".adt", ".adts", ".caf", ".snd",
	".ac3", ".aif", ".cda"
}

def extension_is_audio_video( extension ):
	l_extension = extension.lower()
	return l_extension in common_video_extensions or l_extension in common_audio_extensions

def random_uuid():
	random_data = os.urandom( 16 )
	return uuid.UUID( bytes = random_data )

def get_media_length( file_path ):
	try:
		result = subprocess.run(
			[ 'ffprobe', '-v', 'error', '-show_entries', 'format=duration', '-of', 'default=noprint_wrappers=1:nokey=1', file_path ],
			stdout = subprocess.PIPE,
			stderr = subprocess.STDOUT
		)
		duration = float( result.stdout ) * 1000  # Convert to milliseconds
		return int( duration )
	except Exception as e:
		return 0

def save_to_yaml( data, output_file ):
	with open( output_file, 'w' ) as file:
		yaml.dump( data, file, default_flow_style = False )

def add_to_structure( path_parts, file_info, structure ):
	if len( path_parts ) == 1:
		x = {
			'id': file_info[ 'uuid' ],
			'path': file_info[ 'path' ]
		}
		if SAVE_EXTENSION:
			x[ 'extension' ] = file_info[ 'extension' ]
		if SAVE_DURATION:
			x[ 'length' ] = get_media_length( file_info[ 'path' ] )
		structure.append( x )
	else:
		dir_name = path_parts[ 0 ]
		dir_entry = next( ( item for item in structure if isinstance( item , dict ) and dir_name in item ) , None )
		if dir_entry is None:
			new_dir = { dir_name: [] }
			structure.append( new_dir )
			add_to_structure( path_parts[ 1: ] , file_info , new_dir[ dir_name ] )
		else:
			add_to_structure( path_parts[ 1: ] , file_info , dir_entry[ dir_name ] )

def get_files_recursively( base_dir ):
	file_structure = []
	for root , dirs , files in os.walk( base_dir ):
		dirs[:] = humansorted( dirs )
		files = humansorted( files )
		rel_dir = os.path.relpath( root, base_dir )
		sub_dirs = rel_dir.split( os.sep ) if rel_dir != '.' else []
		for file in files:
			file_path = os.path.join( root, file )
			extension = os.path.splitext( file )[ 1 ]
			if not extension_is_audio_video( extension ):
				continue
			file_info = {
				'uuid': str( random_uuid() ) ,
				'path': os.path.abspath( file_path ) ,
			}
			if SAVE_EXTENSION:
				file_info[ 'extension' ] = extension
			if SAVE_DURATION:
				file_info[ 'length' ] = get_media_length( file_path )
			add_to_structure( sub_dirs + [ file ] , file_info , file_structure )
	return file_structure

if __name__ == "__main__":
	base_directory = sys.argv[ 1 ] if len( sys.argv ) > 1 else os.getcwd()
	base_directory_posix = Path( base_directory )
	output_yaml_file = base_directory_posix.joinpath( f"{base_directory_posix.stem}.yaml" )

	file_structure = get_files_recursively( base_directory )
	save_to_yaml( file_structure , output_yaml_file )

	print( f"YAML file has been created at {output_yaml_file}" )