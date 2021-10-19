/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package ndi

import (
	"fmt"
	"math"
	"reflect"
	"syscall"
	"unsafe"
)

func goStringFromConst(p uintptr) string {
	var len int
	for n := p; *(*byte)(unsafe.Pointer(n)) != 0; n++ {
		len++
	}

	h := &reflect.SliceHeader{uintptr(unsafe.Pointer(p)), len, len + 1}
	return string(*(*[]byte)(unsafe.Pointer(h)))
}

func goStringFromCString(p uintptr) string {
	s := ""
	for ; *(*byte)(unsafe.Pointer(p)) != 0; p++ {
		s = fmt.Sprintf("%s%c", s, *(*byte)(unsafe.Pointer(p)))
	}
	return s
}

type Error struct {
	syscall.Errno
}

func (e *Error) Timeout() bool {
	return e.Errno.Timeout() || uintptr(e.Errno) == 1460
}

type FrameFormat int32

const (
	FrameFormatInterleaved FrameFormat = iota //A fielded frame with the field 0 being on the even lines and field 1 being on the odd lines.
	FrameFormatProgressive                    //A progressive frame.

	//Individual fields.
	FrameFormatField0
	FrameFormatField1
)

const (
	SendTimecodeSynthesize int64 = math.MaxInt64
	SendTimecodeEmpty      int64 = 0
)

type RecvBandwidth int32

const (
	RecvBandwidthMetadataOnly RecvBandwidth = -10 //Receive metadata.
	RecvBandwidthAudioOnly    RecvBandwidth = 10  //Receive metadata, audio.
	RecvBandwidthLowest       RecvBandwidth = 0   //Receive metadata, audio, video at a lower bandwidth and resolution.
	RecvBandwidthHighest      RecvBandwidth = 100 //Receive metadata, audio, video at full resolution.
)

var (
	FourCCTypeUYVY = [4]byte{'U', 'Y', 'V', 'Y'}

	//BGRA
	FourCCTypeBGRA = [4]byte{'B', 'G', 'R', 'A'}
	FourCCTypeBGRX = [4]byte{'B', 'G', 'R', 'X'}

	//This is a UYVY buffer followed immediately by an alpha channel buffer.
	//If the stride of the YCbCr component is "stride", then the alpha channel
	//starts at image_ptr + yres*stride. The alpha channel stride is stride/2.
	FourCCTypeUYVA = [4]byte{'U', 'Y', 'V', 'A'}
)

type RecvColorFormat int32

const (
	RecvColorFormatBGRXBGRA RecvColorFormat = 0 //No alpha channel: BGRX, Alpha channel: BGRA
	RecvColorFormatUYVYBGRA RecvColorFormat = 1 //No alpha channel: UYVY, Alpha channel: BGRA
	RecvColorFormatRGBXRGBA RecvColorFormat = 2 //No alpha channel: RGBX, Alpha channel: RGBA
	RecvColorFormatUYVYRGBA RecvColorFormat = 3 //No alpha channel: UYVY, Alpha channel: RGBA

	//Read the SDK documentation to understand the pros and cons of this format.
	RecvColorFormatFastest RecvColorFormat = 100
)

type FrameType int32

//An enumeration to specify the type of a packet returned by the functions
const (
	FrameTypeNone FrameType = iota
	FrameTypeVideo
	FrameTypeAudio
	FrameTypeMetadata
	FrameTypeError

	//This indicates that the settings on this input have changed.
	//For instamce, this value will be returned from NDIlib_recv_capture_v2 and NDIlib_recv_capture
	//when the device is known to have new settings, for instance the web-url has changed ot the device
	//is now known to be a PTZ camera.
	FrameTypeStatusChange FrameType = 100
)

func NewVideoFrameV2() *VideoFrameV2 {
	vf := &VideoFrameV2{}
	vf.SetDefault()
	return vf
}

//This describes a video frame.
type VideoFrameV2 struct {
	Xres, Yres int32   //The resolution of this frame.
	FourCC     [4]byte //What FourCC this is with. This can be two values.

	//What is the frame-rate of this frame.
	//For instance NTSC is 30000,1001 = 30000/1001 = 29.97fps.
	FrameRateN, FrameRateD int32

	//What is the picture aspect ratio of this frame.
	//For instance 16.0/9.0 = 1.778 is 16:9 video
	//0 means square pixels.
	PictureAspectRatio float32

	//Is this a fielded frame, or is it progressive.
	FrameFormatType FrameFormat

	//The timecode of this frame in 100ns intervals.
	Timecode int64

	//The video data itself.
	Data *byte

	//The inter line stride of the video data, in bytes.
	LineStride int32

	//Per frame metadata for this frame. This is a NULL terminated UTF8 string that should be
	//in XML format. If you do not want any metadata then you may specify NULL here.
	Metadata *byte

	//This is only valid when receiving a frame and is specified as a 100ns time that was the exact
	//moment that the frame was submitted by the sending side and is generated by the SDK. If this
	//value is NDIlib_recv_timestamp_undefined then this value is not available and is NDIlib_recv_timestamp_undefined.
	Timestamp int64
}

func (vf *VideoFrameV2) SetDefault() {
	vf.Xres = 0
	vf.Yres = 0
	vf.FourCC = FourCCTypeUYVA
	vf.FrameRateN = 30000
	vf.FrameRateD = 1001
	vf.PictureAspectRatio = 0
	vf.FrameFormatType = FrameFormatProgressive
	vf.Timecode = SendTimecodeSynthesize
	vf.Data = nil
	vf.LineStride = 0
	vf.Metadata = nil
	vf.Timestamp = SendTimecodeEmpty
}

func (vf *VideoFrameV2) ReadData() []byte {
	v := (*[1920 * 1080 * 4]byte)(unsafe.Pointer(vf.Data)) // Read
	b := v[:vf.LineStride]
	return b
}

func NewAudioFrameV2() *AudioFrameV2 {
	af := &AudioFrameV2{}
	af.SetDefault()
	return af
}

type AudioFrameV2 struct {
	SampleRate, //The sample-rate of this buffer.
	NumChannels, //The number of audio channels.
	NumSamples int32 //The number of audio samples per channel.
	Timecode      int64    //The timecode of this frame in 100ns intervals.
	Data          *float32 //The audio data
	ChannelStride int32    //The inter channel stride of the audio channels, in bytes.

	//Per frame metadata for this frame. This is a NULL terminated UTF8 string that should be
	//in XML format. If you do not want any metadata then you may specify NULL here.
	Metadata *byte

	// This is only valid when receiving a frame and is specified as a 100ns time that was the exact
	// moment that the frame was submitted by the sending side and is generated by the SDK. If this
	// value is NDIlib_recv_timestamp_undefined then this value is not available and is NDIlib_recv_timestamp_undefined.
	Timestamp int64
}

func (af *AudioFrameV2) SetDefault() {
	af.SampleRate = 48000
	af.NumChannels = 2
	af.NumSamples = 0
	af.Timecode = SendTimecodeSynthesize
	af.Data = nil
	af.ChannelStride = 0
	af.Metadata = nil
	af.Timestamp = SendTimecodeEmpty
}

func NewRecvCreateSettings() *RecvCreateSettings {
	s := &RecvCreateSettings{}
	s.SetDefault()
	return s
}

type RecvCreateSettings struct {
	SourceToConnectTo Source

	//Your preference of color space.
	ColorFormat RecvColorFormat

	//The bandwidth setting that you wish to use for this video source. Bandwidth
	//controlled by changing both the compression level and the resolution of the source.
	//A good use for low bandwidth is working on WIFI connections.
	Bandwidth RecvBandwidth

	//When this flag is FALSE, all video that you receive will be progressive. For sources
	//that provide fields, this is de-interlaced on the receiving side (because we cannot change
	//what the up-stream source was actually rendering. This is provided as a convenience to
	//down-stream sources that do not wish to understand fielded video. There is almost no
	//performance impact of using this function.
	AllowVideoFields bool
}

func (s *RecvCreateSettings) SetDefault() {
	s.SourceToConnectTo = Source{}
	s.ColorFormat = RecvColorFormatUYVYBGRA
	s.Bandwidth = RecvBandwidthHighest
	s.AllowVideoFields = true
}

func NewMetadataFrame() *MetadataFrame {
	mf := &MetadataFrame{}
	mf.SetDefault()
	return mf
}

//The data description for metadata
type MetadataFrame struct {
	//The length of the string in UTF8 characters. This includes the NULL terminating character.
	//If this is 0, then the length is assume to be the length of a null terminated string.
	Length int32

	Timecode int64 //The timecode of this frame in 100ns intervals.
	Data     *byte //The metadata as a UTF8 XML string. This is a NULL terminated string.
}

func (mf *MetadataFrame) SetDefault() {
	mf.Length = 0
	mf.Timecode = SendTimecodeSynthesize
	mf.Data = nil
}

//This is a private struct!
type ndiLIBv5 struct {
	// V1.5
	NDIlibInitialize, //bool(*NDIlib_initialize)(void)
	NDIlibDestroy, //void(*NDIlib_destroy)(void)
	NDIlibVersion, //const char* (*NDIlib_version)(void)
	NDIlibIsSupportedCPU, //bool(*NDIlib_is_supported_CPU)(void)
	NDIlibFindCreate, //PROCESSINGNDILIB_DEPRECATED NDIlib_find_instance_t(*NDIlib_find_create)(const NDIlib_find_create_t* p_create_settings)
	NDIlibFindCreateV2, //NDIlib_find_instance_t(*NDIlib_find_create_v2)(const NDIlib_find_create_t* p_create_settings)
	NDIlibFindDestroy, //void(*NDIlib_find_destroy)(NDIlib_find_instance_t p_instance)
	NDIlibFindGetSources, //const NDIlib_source_t* (*NDIlib_find_get_sources)(NDIlib_find_instance_t p_instance, uint32_t* p_no_sources, uint32_t timeout_in_ms)
	NDIlibSendCreate, //NDIlib_send_instance_t(*NDIlib_send_create)(const NDIlib_send_create_t* p_create_settings)
	NDIlibSendDestroy, //void(*NDIlib_send_destroy)(NDIlib_send_instance_t p_instance)
	NDIlibSendSendVideo, //PROCESSINGNDILIB_DEPRECATED void(*NDIlib_send_send_video)(NDIlib_send_instance_t p_instance, const NDIlib_video_frame_t* p_video_data)
	NDIlibSendSendVideoAsync, //PROCESSINGNDILIB_DEPRECATED void(*NDIlib_send_send_video_async)(NDIlib_send_instance_t p_instance, const NDIlib_video_frame_t* p_video_data)
	NDIlibSendSendAudio, //PROCESSINGNDILIB_DEPRECATED void(*NDIlib_send_send_audio)(NDIlib_send_instance_t p_instance, const NDIlib_audio_frame_t* p_audio_data)
	NDIlibSendSendMetadata, //void(*NDIlib_send_send_metadata)(NDIlib_send_instance_t p_instance, const NDIlib_metadata_frame_t* p_metadata)
	NDIlibSendCapture, //NDIlib_frame_type_e(*NDIlib_send_capture)(NDIlib_send_instance_t p_instance, NDIlib_metadata_frame_t* p_metadata, uint32_t timeout_in_ms)
	NDIlibSendFreeMetadata, //void(*NDIlib_send_free_metadata)(NDIlib_send_instance_t p_instance, const NDIlib_metadata_frame_t* p_metadata)
	NDIlibSendGetTally, //bool(*NDIlib_send_get_tally)(NDIlib_send_instance_t p_instance, NDIlib_tally_t* p_tally, uint32_t timeout_in_ms)
	NDIlibSendGetNoConnections, //int(*NDIlib_send_get_no_connections)(NDIlib_send_instance_t p_instance, uint32_t timeout_in_ms)
	NDIlibSendClearConnectionMetadata, //void(*NDIlib_send_clear_connection_metadata)(NDIlib_send_instance_t p_instance)
	NDIlibSendAddConnectionMetadata, //void(*NDIlib_send_add_connection_metadata)(NDIlib_send_instance_t p_instance, const NDIlib_metadata_frame_t* p_metadata)
	NDIlibSendSetFailover, //void(*NDIlib_send_set_failover)(NDIlib_send_instance_t p_instance, const NDIlib_source_t* p_failover_source)
	NDIlibRecvCreateV2, //NDIlib_recv_instance_t(*NDIlib_recv_create_v2)(const NDIlib_recv_create_t* p_create_settings)
	NDIlibRecvCreate, //NDIlib_recv_instance_t(*NDIlib_recv_create)(const NDIlib_recv_create_t* p_create_settings)
	NDIlibRecvDestroy, //void(*NDIlib_recv_destroy)(NDIlib_recv_instance_t p_instance)
	NDIlibRecvCapture, //PROCESSINGNDILIB_DEPRECATED NDIlib_frame_type_e(*NDIlib_recv_capture)(NDIlib_recv_instance_t p_instance, NDIlib_video_frame_t* p_video_data, NDIlib_audio_frame_t* p_audio_data, NDIlib_metadata_frame_t* p_metadata, uint32_t timeout_in_ms)
	NDIlibRecvFreeVideo, //PROCESSINGNDILIB_DEPRECATED void(*NDIlib_recv_free_video)(NDIlib_recv_instance_t p_instance, const NDIlib_video_frame_t* p_video_data)
	NDIlibRecvFreeAudio, //PROCESSINGNDILIB_DEPRECATED void(*NDIlib_recv_free_audio)(NDIlib_recv_instance_t p_instance, const NDIlib_audio_frame_t* p_audio_data)
	NDIlibRecvFreeMetadata, //void(*NDIlib_recv_free_metadata)(NDIlib_recv_instance_t p_instance, const NDIlib_metadata_frame_t* p_metadata)
	NDIlibRecvSendMetadata, //bool(*NDIlib_recv_send_metadata)(NDIlib_recv_instance_t p_instance, const NDIlib_metadata_frame_t* p_metadata)
	NDIlibRecvSetTally, //bool(*NDIlib_recv_set_tally)(NDIlib_recv_instance_t p_instance, const NDIlib_tally_t* p_tally)
	NDIlibRecvGetPerformance, //void(*NDIlib_recv_get_performance)(NDIlib_recv_instance_t p_instance, NDIlib_recv_performance_t* p_total, NDIlib_recv_performance_t* p_dropped)
	NDIlibRecvGetQueue, //void(*NDIlib_recv_get_queue)(NDIlib_recv_instance_t p_instance, NDIlib_recv_queue_t* p_total)
	NDIlibRecvClearConnectionMetadata, //void(*NDIlib_recv_clear_connection_metadata)(NDIlib_recv_instance_t p_instance)
	NDIlibRecvAddConnectionMetadata, //void(*NDIlib_recv_add_connection_metadata)(NDIlib_recv_instance_t p_instance, const NDIlib_metadata_frame_t* p_metadata)
	NDIlibRecvGetNoConnections, //int(*NDIlib_recv_get_no_connections)(NDIlib_recv_instance_t p_instance)
	NDIlibRoutingCreate, //NDIlib_routing_instance_t(*NDIlib_routing_create)(const NDIlib_routing_create_t* p_create_settings)
	NDIlibRoutingDestroy, //void(*NDIlib_routing_destroy)(NDIlib_routing_instance_t p_instance)
	NDIlibRoutingChange, //bool(*NDIlib_routing_change)(NDIlib_routing_instance_t p_instance, const NDIlib_source_t* p_source)
	NDIlibRoutingClear, //bool(*NDIlib_routing_clear)(NDIlib_routing_instance_t p_instance)
	NDIlibUtilSendSendAudioInterleaved16s, //void(*NDIlib_util_send_send_audio_interleaved_16s)(NDIlib_send_instance_t p_instance, const NDIlib_audio_frame_interleaved_16s_t* p_audio_data)
	NDIlibUtilAudioToInterleaved16s, //PROCESSINGNDILIB_DEPRECATED void(*NDIlib_util_audio_to_interleaved_16s)(const NDIlib_audio_frame_t* p_src, NDIlib_audio_frame_interleaved_16s_t* p_dst)
	NDIlibUtilAudioFromInterleaved16s, //PROCESSINGNDILIB_DEPRECATED void(*NDIlib_util_audio_from_interleaved_16s)(const NDIlib_audio_frame_interleaved_16s_t* p_src, NDIlib_audio_frame_t* p_dst)

	// V2
	NDIlibFindWaitForSources, //bool(*NDIlib_find_wait_for_sources)(NDIlib_find_instance_t p_instance, uint32_t timeout_in_ms)
	NDIlibFindGetCurrentSources, //const NDIlib_source_t* (*NDIlib_find_get_current_sources)(NDIlib_find_instance_t p_instance, uint32_t* p_no_sources)
	NDIlibUtilAudioToInterleaved32f, //PROCESSINGNDILIB_DEPRECATED void(*NDIlib_util_audio_to_interleaved_32f)(const NDIlib_audio_frame_t* p_src, NDIlib_audio_frame_interleaved_32f_t* p_dst)
	NDIlibUtilAudioFromInterleaved32f, //PROCESSINGNDILIB_DEPRECATED void(*NDIlib_util_audio_from_interleaved_32f)(const NDIlib_audio_frame_interleaved_32f_t* p_src, NDIlib_audio_frame_t* p_dst)
	NDIlibUtilSendSendAudioInterleaved32f, //void(*NDIlib_util_send_send_audio_interleaved_32f)(NDIlib_send_instance_t p_instance, const NDIlib_audio_frame_interleaved_32f_t* p_audio_data)

	// V3
	NDIlibRecvFreeVideoV2, //void(*NDIlib_recv_free_video_v2)(NDIlib_recv_instance_t p_instance, const NDIlib_video_frame_v2_t* p_video_data)
	NDIlibRecvFreeAudioV2, //void(*NDIlib_recv_free_audio_v2)(NDIlib_recv_instance_t p_instance, const NDIlib_audio_frame_v2_t* p_audio_data)
	NDIlibRecvCaptureV2, //NDIlib_frame_type_e(*NDIlib_recv_capture_v2)(NDIlib_recv_instance_t p_instance, NDIlib_video_frame_v2_t* p_video_data, NDIlib_audio_frame_v2_t* p_audio_data, NDIlib_metadata_frame_t* p_metadata, uint32_t timeout_in_ms)
	NDIlibSendSendVideoV2, //void(*NDIlib_send_send_video_v2)(NDIlib_send_instance_t p_instance, const NDIlib_video_frame_v2_t* p_video_data)
	NDIlibSendSendVideoAsyncV2, //void(*NDIlib_send_send_video_async_v2)(NDIlib_send_instance_t p_instance, const NDIlib_video_frame_v2_t* p_video_data)
	NDIlibSendSendAudioV2, //void(*NDIlib_send_send_audio_v2)(NDIlib_send_instance_t p_instance, const NDIlib_audio_frame_v2_t* p_audio_data)
	NDIlibUtilAudioToInterleaved16sV2, //void(*NDIlib_util_audio_to_interleaved_16s_v2)(const NDIlib_audio_frame_v2_t* p_src, NDIlib_audio_frame_interleaved_16s_t* p_dst)
	NDIlibUtilAudioFromInterleaved16sV2, //void(*NDIlib_util_audio_from_interleaved_16s_v2)(const NDIlib_audio_frame_interleaved_16s_t* p_src, NDIlib_audio_frame_v2_t* p_dst)
	NDIlibUtilAudioToInterleaved32fV2, //void(*NDIlib_util_audio_to_interleaved_32f_v2)(const NDIlib_audio_frame_v2_t* p_src, NDIlib_audio_frame_interleaved_32f_t* p_dst)
	NDIlibUtilAudioFromInterleaved32fV2, //void(*NDIlib_util_audio_from_interleaved_32f_v2)(const NDIlib_audio_frame_interleaved_32f_t* p_src, NDIlib_audio_frame_v2_t* p_dst)

	// V3.01
	NDIlibRecvFreeString, //void(*NDIlib_recv_free_string)(NDIlib_recv_instance_t p_instance, const char* p_string)
	NDIlibRecvPtzIsSupported, //bool(*NDIlib_recv_ptz_is_supported)(NDIlib_recv_instance_t p_instance)
	NDIlibRecvRecordingIsSupported, //bool(*NDIlib_recv_recording_is_supported)(NDIlib_recv_instance_t p_instance)
	NDIlibRecvGetWebControl, //const char*(*NDIlib_recv_get_web_control)(NDIlib_recv_instance_t p_instance)
	NDIlibRecvPtzZoom, //bool(*NDIlib_recv_ptz_zoom)(NDIlib_recv_instance_t p_instance, const float zoom_value)
	NDIlibRecvPtzZoomSpeed, //bool(*NDIlib_recv_ptz_zoom_speed)(NDIlib_recv_instance_t p_instance, const float zoom_speed)
	NDIlibRecvPtzPanTilt, //bool(*NDIlib_recv_ptz_pan_tilt)(NDIlib_recv_instance_t p_instance, const float pan_value, const float tilt_value)
	NDIlibRecvPtzPanTiltSpeed, //bool(*NDIlib_recv_ptz_pan_tilt_speed)(NDIlib_recv_instance_t p_instance, const float pan_speed, const float tilt_speed)
	NDIlibRecvPtzStorePreset, //bool(*NDIlib_recv_ptz_store_preset)(NDIlib_recv_instance_t p_instance, const int preset_no)
	NDIlibRecvPtzRecallPreset, //bool(*NDIlib_recv_ptz_recall_preset)(NDIlib_recv_instance_t p_instance, const int preset_no, const float speed)
	NDIlibRecvPtzAutoFocus, //bool(*NDIlib_recv_ptz_auto_focus)(NDIlib_recv_instance_t p_instance)
	NDIlibRecvPtzFocus, //bool(*NDIlib_recv_ptz_focus)(NDIlib_recv_instance_t p_instance, const float focus_value)
	NDIlibRecvPtzFocusSpeed, //bool(*NDIlib_recv_ptz_focus_speed)(NDIlib_recv_instance_t p_instance, const float focus_speed)
	NDIlibRecvPtzWhiteBalanceAuto, //bool(*NDIlib_recv_ptz_white_balance_auto)(NDIlib_recv_instance_t p_instance)
	NDIlibRecvPtzWhiteBalanceIndoor, //bool(*NDIlib_recv_ptz_white_balance_indoor)(NDIlib_recv_instance_t p_instance)
	NDIlibRecvPtzWhiteBalanceOutdoor, //bool(*NDIlib_recv_ptz_white_balance_outdoor)(NDIlib_recv_instance_t p_instance)
	NDIlibRecvPtzWhiteBalanceOneshot, //bool(*NDIlib_recv_ptz_white_balance_oneshot)(NDIlib_recv_instance_t p_instance)
	NDIlibRecvPtzWhiteBalanceManual, //bool(*NDIlib_recv_ptz_white_balance_manual)(NDIlib_recv_instance_t p_instance, const float red, const float blue)
	NDIlibRecvPtzExposureAuto, //bool(*NDIlib_recv_ptz_exposure_auto)(NDIlib_recv_instance_t p_instance)
	NDIlibRecvPtzExposureManual, //bool(*NDIlib_recv_ptz_exposure_manual)(NDIlib_recv_instance_t p_instance, const float exposure_level)
	NDIlibRecvRecordingStart, //bool(*NDIlib_recv_recording_start)(NDIlib_recv_instance_t p_instance, const char* p_filename_hint)
	NDIlibRecvRecordingStop, //bool(*NDIlib_recv_recording_stop)(NDIlib_recv_instance_t p_instance)
	NDIlibRecvRecordingSetAudioLevel, //bool(*NDIlib_recv_recording_set_audio_level)(NDIlib_recv_instance_t p_instance, const float level_dB)
	NDIlibRecvRecordingIsRecording, //bool(*NDIlib_recv_recording_is_recording)(NDIlib_recv_instance_t p_instance)
	NDIlibRecvRecordingGetFilename, //const char*(*NDIlib_recv_recording_get_filename)(NDIlib_recv_instance_t p_instance)
	NDIlibRecvRecordingGetError, //const char*(*NDIlib_recv_recording_get_error)(NDIlib_recv_instance_t p_instance)
	NDIlibRecvRecordingGetTimes, //bool(*NDIlib_recv_recording_get_times)(NDIlib_recv_instance_t p_instance, NDIlib_recv_recording_time_t* p_times)

	// V3.1
	NDIlibRecvInstanceT, //NDIlib_recv_instance_t(*recv_create_v3)(const NDIlib_recv_create_v3_t* p_create_settings)

	// V3.5
	NDIlibRecvConnect, //void(*recv_connect)(NDIlib_recv_instance_t p_instance, const NDIlib_source_t* p_src)

	// V3.6
	NDIlibFramesyncInstanceT, //NDIlib_framesync_instance_t(*framesync_create)(NDIlib_recv_instance_t p_receiver)
	NDIlibFramesyncDestroy, // void(*framesync_destroy)(NDIlib_framesync_instance_t p_instance)
	NDIlibFramesyncCaptureAudio, //void(*framesync_capture_audio)(NDIlib_framesync_instance_t p_instance, NDIlib_audio_frame_v2_t* p_audio_data, int sample_rate, int no_channels, int no_samples)
	NDIlibFramesyncFreeAudio, //void(*framesync_free_audio)(NDIlib_framesync_instance_t p_instance, NDIlib_audio_frame_v2_t* p_audio_data)
	NDIlibFramesyncCaptureVideo, //void(*framesync_capture_video)(NDIlib_framesync_instance_t p_instance, NDIlib_video_frame_v2_t* p_video_data, NDIlib_frame_format_type_e field_type)
	NDIlibFramesyncFreeVideo, //void(*framesync_free_video)(NDIlib_framesync_instance_t p_instance, NDIlib_video_frame_v2_t* p_video_data)
	NDIlibUtilSendSendAudioInterleaved32s, //void(*util_send_send_audio_interleaved_32s)(NDIlib_send_instance_t p_instance, const NDIlib_audio_frame_interleaved_32s_t* p_audio_data)
	NDIlibUtilAudioToInterleaved32sV2, //void(*util_audio_to_interleaved_32s_v2)(const NDIlib_audio_frame_v2_t* p_src, NDIlib_audio_frame_interleaved_32s_t* p_dst)
	NDIlibUtilAudioFromInterleaved32sV2, //void(*util_audio_from_interleaved_32s_v2)(const NDIlib_audio_frame_interleaved_32s_t* p_src, NDIlib_audio_frame_v2_t* p_dst)

	// V3.8
	NDIlibSourceTv38, //const NDIlib_source_t* (*send_get_source_name)(NDIlib_send_instance_t p_instance)

	// V4.0
	NDIlibSendSendAudioV3, //void(*send_send_audio_v3)(NDIlib_send_instance_t p_instance, const NDIlib_audio_frame_v3_t* p_audio_data)
	NDIlibUtilV210ToP216, //void(*util_V210_to_P216)(const NDIlib_video_frame_v2_t* p_src_v210, NDIlib_video_frame_v2_t* p_dst_p216)
	NDIlibUtilP216ToV210, //void(*util_P216_to_V210)(const NDIlib_video_frame_v2_t* p_src_p216, NDIlib_video_frame_v2_t* p_dst_v210)

	// V4.1
	NDIlibRoutingGetNoConnections, //int (*routing_get_no_connections)(NDIlib_routing_instance_t p_instance, uint32_t timeout_in_ms)
	NDIlibSourceT, //const NDIlib_source_t* (*routing_get_source_name)(NDIlib_routing_instance_t p_instance)
	NDIlibFrameTypeE, // NDIlib_frame_type_e(*recv_capture_v3)(NDIlib_recv_instance_t p_instance, NDIlib_video_frame_v2_t* p_video_data, NDIlib_audio_frame_v3_t* p_audio_data, NDIlib_metadata_frame_t* p_metadata, uint32_t timeout_in_ms);             // The amount of time in milliseconds to wait for data
	NDIlibRecvFreeAudioV3, //void(*recv_free_audio_v3)(NDIlib_recv_instance_t p_instance, const NDIlib_audio_frame_v3_t* p_audio_data)
	NDIlibFramesyncCaptureAudioV2, // void(*framesync_capture_audio_v2)(NDIlib_framesync_instance_t p_instance, NDIlib_audio_frame_v3_t* p_audio_data, int sample_rate, int no_channels, int no_samples)
	NDIlibFramesyncFreeAudioV2, // void(*framesync_free_audio_v2)(NDIlib_framesync_instance_t p_instance, NDIlib_audio_frame_v3_t* p_audio_data)
	NDIlibFramesyncAudioQueueDepth, // int(*framesync_audio_queue_depth)(NDIlib_framesync_instance_t p_instance)

	// v4.5
	NDIlibRecvPtzExposureManualV2 uintptr //bool(*recv_ptz_exposure_manual_v2)(NDIlib_framesync_instance_t p_instance, const float iris, const float gain, const float shutter_speed)
}
