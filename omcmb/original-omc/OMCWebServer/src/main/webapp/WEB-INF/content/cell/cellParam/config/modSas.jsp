<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	
	<%-- 下面是新增的参数 --%>
	<li class="paramNodesUl">
		<label for="SAS_ENABLE_MODE_name">${SAS_ENABLE_MODE_name}</label>
        <select id="SAS_ENABLE_MODE_name" name="SAS_ENABLE_MODE" class="border border-box" onblur="createMML();">
            <option value=""></option>
            <option value="1">Enable</option>
            <option value="0">Disable</option>
        </select>
	</li>
	<%-- <li class="paramNodesUl">
		<label for="LTE_SAS_CBSD_SN_name">${LTE_SAS_CBSD_SN_name}</label>
		<input id="LTE_SAS_CBSD_SN_name" name="LTE_SAS_CBSD_SN" title="${LTE_SAS_CBSD_SN_title}" class="border border-box" 
		 min_value="0" max_value="64" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_SAS_CBSD_SN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_SAS_CBSD_SN_title}
		</div>
	</li> --%>
	<li class="paramNodesUl">
		<label for="CBSD_CATEGORY_name">${CBSD_CATEGORY_name}</label>
        <select id="CBSD_CATEGORY_name" name="CBSD_CATEGORY" class="border border-box" onblur="createMML();">
            <option value=""></option>
            <option value="A">A</option>
            <option value="B">B</option>
        </select>
	</li>
	<li class="paramNodesUl">
		<label for="UR_ID_name">${UR_ID_name}</label>
		<input id="UR_ID_name" name="UR_ID" title="${UR_ID_title }" class="border border-box" 
		 min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);createMML();"/>
		<div id="UR_ID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${UR_ID_title}
		</div>
	</li>
	<li class="paramNodesUl">
		<label for="FCCID_name">${FCCID_name }</label>
        <input id="FCCID_name" name=FCCID_ title="${FCCID_title}" class="border border-box" 
         min_length="0" max_length="19" onblur="validateMaxAndMinLength(event);createMML();"/>
        <div id="FCCID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${FCCID_title}
        </div>
	</li>
<%-- 	<li class="paramNodesUl">
		<label for="LTE_SAS_SUPPORTSPEC_name">${LTE_SAS_SUPPORTSPEC_name}</label>
		<input id="LTE_SAS_SUPPORTSPEC_name" name="LTE_SAS_SUPPORTSPEC" type="text" onblur="validateMaxAndMinLength(event);createMML();" title="${LTE_SAS_SUPPORTSPEC_title}" 
			min_length="0" max_length="64" class="border border-box"/>
		<div id="LTE_SAS_SUPPORTSPEC_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_SAS_SUPPORTSPEC_title}
		</div>
	</li> --%>
	<li class="paramNodesUl">
		<label for="GPS_LATITUDE_name">${GPS_LATITUDE_name}</label>
		<input id="GPS_LATITUDE_name" name="GPS_LATITUDE" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${GPS_LATITUDE_title}" 
			min_value="-90000000" max_value="90000000" class="border border-box"/>
		<div id="GPS_LATITUDE_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${GPS_LATITUDE_title}
		</div>
	</li>
	<li class="paramNodesUl">
        <label for="GPS_LONGITUDE_name">${GPS_LONGITUDE_name}</label>
        <input id="GPS_LONGITUDE_name" name=GPS_LONGITUDE type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${GPS_LONGITUDE_title}" 
            min_value="-180000000" max_value="180000000" class="border border-box"/>
        <div id="GPS_LONGITUDE_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${GPS_LONGITUDE_title}
        </div>
    </li>
    <li class="paramNodesUl">
        <label for="ANT_HEIGHT_name">${ANT_HEIGHT_name}</label>
        <input id="ANT_HEIGHT_name" name="ANT_HEIGHT" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${ANT_HEIGHT_title}" 
            min_value="0" max_value="300" class="border border-box"/>
        <div id="ANT_HEIGHT_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${ANT_HEIGHT_title}
        </div>
    </li>
<%--     <li class="paramNodesUl">
        <label for="ANT_HEIGHT_TYPE_name">${ANT_HEIGHT_TYPE_name}</label>
        <select id="ANT_HEIGHT_TYPE_name" name="ANT_HEIGHT_TYPE" class="border border-box" onblur="createMML();">
            <option value=""></option>
            <option value="AGL">AGL</option>
        </select>
    </li> --%>
    <li class="paramNodesUl">
        <label for="INDOOR_DEPLOYMENT_name">${INDOOR_DEPLOYMENT_name}</label>
        <select id="INDOOR_DEPLOYMENT_name" name="INDOOR_DEPLOYMENT" class="border border-box" onblur="createMML();">
            <option value=""></option>
            <option value="1">Indoor</option>
            <option value="0">Outdoor</option>
        </select>
    </li>   
    <li class="paramNodesUl">
        <label for="ANT_GAIN_name">${ANT_GAIN_name}</label>
        <input id="ANT_GAIN_name" name="ANT_GAIN" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${ANT_GAIN_title}" 
            min_value="-5" max_value="30" class="border border-box"/>
        <div id="ANT_GAIN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${ANT_GAIN_title}
        </div>
    </li>    
    <li class="paramNodesUl">
        <label for="CFG_LOW_FREQUENCY_name">${CFG_LOW_FREQUENCY_name}</label>
        <input id="CFG_LOW_FREQUENCY_name" name="CFG_LOW_FREQUENCY" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${CFG_LOW_FREQUENCY_title}" 
            min_value="3550" max_value="3700" class="border border-box"/>
        <div id="CFG_LOW_FREQUENCY_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${CFG_LOW_FREQUENCY_title}
        </div>
    </li>
    <li class="paramNodesUl">
        <label for="CFG_HIGH_FREQUENCY_name">${CFG_HIGH_FREQUENCY_name}</label>
        <input id="CFG_HIGH_FREQUENCY_name" name="CFG_HIGH_FREQUENCY" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${CFG_HIGH_FREQUENCY_title}" 
            min_value="3550" max_value="3700" class="border border-box"/>
        <div id="CFG_HIGH_FREQUENCY_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${CFG_HIGH_FREQUENCY_title}
        </div>
    </li>     
    <li class="paramNodesUl">
        <label for="CFG_MAX_PSD_name">${CFG_MAX_PSD_name}</label>
        <input id="CFG_MAX_PSD_name" name="CFG_MAX_PSD" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${CFG_MAX_PSD_title}" 
            min_value="0" max_value="37" class="border border-box"/>
        <div id="CFG_MAX_PSD_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${CFG_MAX_PSD_title}
        </div>
    </li>     
    <li class="paramNodesUl">
        <label for="ANT_AZIMUTH_name">${ANT_AZIMUTH_name}</label>
        <input id="ANT_AZIMUTH_name" name="ANT_AZIMUTH" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${ANT_AZIMUTH_title}" 
            min_value="0" max_value="359" class="border border-box"/>
        <div id="ANT_AZIMUTH_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${ANT_AZIMUTH_title}
        </div>
    </li>     
    <li class="paramNodesUl">
        <label for="ANT_DOWNTILT_name">${ANT_DOWNTILT_name}</label>
        <input id="ANT_DOWNTILT_name" name="ANT_DOWNTILT" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${ANT_DOWNTILT_title}" 
            min_value="-90" max_value="90" class="border border-box"/>
        <div id="ANT_DOWNTILT_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${ANT_DOWNTILT_title}
        </div>
    </li>     
    <li class="paramNodesUl">
        <label for="ANT_BW_name">${ANT_BW_name}</label>
        <input id="ANT_BW_name" name="ANT_BW" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${ANT_BW_title}" 
            min_value="0" max_value="360" class="border border-box"/>
        <div id="ANT_BW_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${ANT_BW_title}
        </div>
    </li>     
    <li class="paramNodesUl">
        <label for="CALLSIGN_name">${CALLSIGN_name}</label>
        <input id="CALLSIGN_name" name="CALLSIGN" type="text" onblur="validateMaxAndMinLength(event);createMML();" title="${CALLSIGN_title}" 
            min_length="0" max_length="256" class="border border-box"/>
        <div id="CALLSIGN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${CALLSIGN_title}
        </div>
    </li>     
    <li class="paramNodesUl">
        <label for="SAS_GROUPTYPE_LIST_name">${SAS_GROUPTYPE_LIST_name}</label>
        <input id="SAS_GROUPTYPE_LIST_name" name="SAS_GROUPTYPE_LIST" type="text" onblur="validateMaxAndMinLength(event);createMML();" title="${SAS_GROUPTYPE_LIST_title}" 
            min_length="0" max_length="128" class="border border-box"/>
        <div id="SAS_GROUPTYPE_LIST_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${SAS_GROUPTYPE_LIST_title}
        </div>
    </li>     
    <li class="paramNodesUl">
        <label for="SAS_GROUPID_LIST_name">${SAS_GROUPID_LIST_name}</label>
        <input id="SAS_GROUPID_LIST_name" name="SAS_GROUPID_LIST" type="text" onblur="validateMaxAndMinLength(event);createMML();" title="${SAS_GROUPID_LIST_title}" 
            min_length="0" max_length="128" class="border border-box"/>
        <div id="SAS_GROUPID_LIST_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${SAS_GROUPID_LIST_title}
        </div>
    </li>    
</ul>

<script type ="text/javascript">

</script>