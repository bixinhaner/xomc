<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>
<%@ taglib uri="http://java.sun.com/jsp/jstl/core" prefix="c"%>

<ul id="paramNodesUl" class="paramNodesUl">
	<c:if test='${hardwareVersion != "NBIOT1.0"}'>
		<li>
			<label for="LTE_GPS_SYNC_ENABLE_name">${LTE_GPS_SYNC_ENABLE_name }</label>
			<select id="LTE_GPS_SYNC_ENABLE_name" name="LTE_GPS_SYNC_ENABLE" onchange="gpsChangeEvent(this)" class="border border-box" onblur="createMML();">
				<option value=""></option>
				<option value="true">true</option>
				<option value="false">false</option>
			</select>
		</li>
		<li style="diplay: none;">
			<label for="LTE_GPS_CONTROL_ENABLE_name">${LTE_GPS_CONTROL_ENABLE_name }</label>
			<select id="LTE_GPS_CONTROL_ENABLE_name" name="LTE_GPS_CONTROL_ENABLE" class="border border-box" onblur="createMML();">
				<option value=""></option>
				<option value="true">true</option>
				<option value="false">false</option>
			</select>
		</li>
		<li style="diplay: none;">
			<label for="LTE_GPS_SYNC_SOURCE_name">${LTE_GPS_SYNC_SOURCE_name }</label>
			<input id="LTE_GPS_SYNC_SOURCE_name" type="hidden" name="LTE_GPS_SYNC_SOURCE"/>
			<select id="LTE_GPS_SYNC_SOURCE_name_show" class="border border-box ignore">
				<option value="1">GPS</option>
				<option value="2">Glonass</option>
				<option value="8">BeiDou</option>
				<option value="16">Galileo</option>
				<option value="32">Qzss</option>
			</select>
		</li>
	</c:if>
	<c:if test='${hardwareVersion == "NBIOT1.0"}'>
		<li>
			<label for="LTE_GPS_SYNC_ENABLE_name">${LTE_GPS_SYNC_ENABLE_name }</label>
			<select id="LTE_GPS_SYNC_ENABLE_name" name="LTE_GPS_SYNC_ENABLE" onchange="gpsChangeEvent(this)" class="border border-box" onblur="createMML();">
				<option value=""></option>
				<option value="1">true</option>
				<option value="0">false</option>
			</select>
		</li>
		<li style="diplay: none;">
			<label for="LTE_GPS_SYNC_SOURCE_name">${LTE_GPS_SYNC_SOURCE_name }</label>
			<input id="LTE_GPS_SYNC_SOURCE_name" type="hidden" name="LTE_GPS_SYNC_SOURCE"/>
			<select id="LTE_GPS_SYNC_SOURCE_name_show" class="border border-box ignore">
				<option value="1">GPS</option>
				<option value="2">Glonass</option>
				<option value="8">BeiDou</option>
				<option value="16">Galileo</option>
				<option value="32">Qzss</option>
			</select>
		</li>
	</c:if>
</ul>

<script type ="text/javascript">
	$('#LTE_GPS_SYNC_SOURCE_name_show').combobox({
		height: 26,
		value: '',
		multiple: true,
		onChange: function(newVal, oldVal){
			var oldlist = oldVal||[],
				newList = newVal||[];
			
			if(newList.includes('2') && !oldlist.includes('2')) {
				var idx = newList.indexOf('8');
				idx>=0 && newList.splice(idx,1);
				
				$(this).combobox('setValues',newList.join(','));
			}
			if(newList.includes('8') && !oldlist.includes('8')) {
				var idx = oldVal.indexOf('2');
				idx>=0 && newList.splice(idx,1);
				
				$(this).combobox('setValues',newList.join(','));
			}
			
			$('#LTE_GPS_SYNC_SOURCE_name').val(newList.join(','));
			
			createMML();
		}
	})
	
	function gpsChangeEvent(dom) {
		var val = $(dom).val();
		try{
			if(val == 'true' || val == '1') {
				$('#LTE_GPS_SYNC_SOURCE_name').parent().show();
				$('#LTE_GPS_CONTROL_ENABLE_name').parent().show();
			}else {
				$('#LTE_GPS_SYNC_SOURCE_name').parent().hide();
				$('#LTE_GPS_CONTROL_ENABLE_name').parent().hide();
			}
		}catch(e){}
	}
</script>