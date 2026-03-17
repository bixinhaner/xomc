<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>
<%@ taglib uri="http://java.sun.com/jsp/jstl/core" prefix="c"%>
<style>
.reSetConfirm + .messager-button{
	padding-top:20px;
}
.ip-item-span {
	font-size: 10px;
	padding: 3px;
	margin: 3px 5px 0 0;
	background: #efefef;
}
.ip-item-span i{
	font-size: 14px;
}
.ip-item-text {
	display: inline-block;
	min-width: 90px;
}
.invalid-ip-tips {
	postion: relative;
}
.invalid-ip-tips::after {
	position: absolute;
	content: 'Need IP format(max: 16,no duplicate)';
	color: #FA5555;
	bottom: -15px;
}
.paramNodesUl li{
	width: 48.5%;
	display: inline-block;
	margin: 6px;
	height: 62px;
	border: 1px;
	vertical-align: top;
}
</style>

<ul id="paramNodesUl" class="paramNodesUl">
	<div style="display: flex;">
		<div id="oneMMEPool" style="max-width: 50%;">
			<li style="position: relative;height: auto;min-width: 260px;">
				<label for="mme1_ip_input">${LTE_X_MMMM_POOL_MME_LIST1_name }</label>
				<input id="mme1_ip_input" title="${LTE_X_MMMM_POOL_MME_LIST1_title }" class="border border-box" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);" js_regex="no_zh"/> 
				<i class="el-icon el-icon-plus" style="position: absolute; left: 210px; top: 23px;" onclick="addMmeIP(1)"></i>
				<div id="LTE_SIGLINK_SERVER_LIST_name_err" class="errSpan" style="display: block;margin-top: 5px;">
					${LTE_SIGLINK_SERVER_LIST_title }
				</div>
				<div id="mme1_ip_ctn" style="display: flex;flex-wrap: wrap; min-width: 240px;"></div>
				<input id="mme1_ip_hidden" type="hidden" name="LTE_X_MMMM_POOL_MME_LIST1" />
		    </li>
			<li>
				<div style="display: flex;">
					<div>
						<label for="mme1_select_control" style="max-width: 225px;min-width: 220px;">${LTE_X_MMMM_MMELIST_TUNNEL_MAP_name }</label>
						<select id="mme1_select_control" title="${LTE_X_MMMM_MMELIST_TUNNEL_MAP_title }" class="border border-box" disabled
							onblur="updateControlPlane(1)" js_regex="no_zh">
							<option value="tunnel1:LTE_POOL_MME_LIST1">Tunnel1</option>
							<option value="" selected>--Unbound--</option>
						</select>
						<div id="mme1_control_tips" class="errSpan" style="display: block;margin-top: 5px;overflow: visible;white-space: nowrap;">
							Control Plane Interface can`t be the same
						</div>
					</div>
					<div style="min-width: 20px;padding: 25px 25px 25px 20px;">--</div>
					<div>
						<label for="mme2_select_control" style="max-width: 210px;">&nbsp;&nbsp;</label>
						<select id="mme2_select_control" title="${LTE_X_MMMM_MMELIST_TUNNEL_MAP_title }" class="border border-box" disabled
							onblur="updateControlPlane(2);" js_regex="no_zh">
							<option value="tunnel2:LTE_POOL_MME_LIST2">Tunnel2</option>
							<option value="" selected>--Unbound--</option>
						</select>
						<div id="mme2_control_tips" class="errSpan" style="display: block;margin-top: 5px;">
							
						</div>
					</div>
				</div>
				<input id="mme_control_hidden" type="hidden" name="LTE_X_MMMM_MMELIST_TUNNEL_MAP" />
		    </li>
			<c:if test='${hardwareVersion != "CR_B4860_SC4.0" && hardwareVersion != "CR_B4860_CA4.0" && hardwareVersion != "CR_B4860_DC4.0" && hardwareVersion != "CR_B4860_TC4.0" && hardwareVersion != "MLN_SC1.0" && hardwareVersion != "MLN_CA1.0" && hardwareVersion != "MLN_DC1.0"}'>
				<li>
					<label for="mme1_select_user" style="max-width: 225px;min-width: 220px;">${LTE_X_MMMM_MMELIST1_UP_ITF_name }</label>
					<select id="mme1_select_user" name="LTE_X_MMMM_MMELIST1_UP_ITF" title="${LTE_X_MMMM_MMELIST1_UP_ITF_title }" class="border border-box" disabled
						onblur="createMML();" js_regex="no_zh">
						<option value="eth2">WAN(DefaultValue)</option>
						<option value="ipsec">ipsec</option>
					</select>
					<div id="mme1_select_user_name_err" class="errSpan" style="display: block;margin-top: 5px;">
						
					</div>
			    </li>
			 </c:if>
		</div>
		
		<div id="twoMMEPool" style="display: block">
			<li style="position: relative;height: auto;min-width: 260px;">
				<label for="mme2_ip_input">${LTE_X_MMMM_POOL_MME_LIST2_name }</label>
				<input id="mme2_ip_input" name="LTE_X_MMMM_POOL_MME_LIST2" title="${LTE_X_MMMM_POOL_MME_LIST2_title }" class="border border-box" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/> 
				<i class="el-icon el-icon-plus" style="position: absolute; left: 210px; top: 23px;" onclick="addMmeIP(2)"></i>
				<div id="LTE_X_MMMM_POOL_MME_LIST1_name_err" class="errSpan" style="display: block;margin-top: 5px;">
					${LTE_SIGLINK_SERVER_LIST_title }
				</div>
				<div id="mme2_ip_ctn" style="display: flex;flex-wrap: wrap; min-width: 240px;"></div>
			</li>
			<li>
				
			</li>
			<c:if test='${hardwareVersion != "CR_B4860_SC4.0" && hardwareVersion != "CR_B4860_CA4.0" && hardwareVersion != "CR_B4860_DC4.0" && hardwareVersion != "CR_B4860_TC4.0" && hardwareVersion != "MLN_SC1.0" && hardwareVersion != "MLN_CA1.0" && hardwareVersion != "MLN_DC1.0"}'>
				<li>
					<label for="mme2_select_user" style="max-width: 225px;min-width: 220px;">${LTE_X_MMMM_MMELIST2_UP_ITF_name }</label>
					<select id="mme2_select_user" name="LTE_X_MMMM_MMELIST2_UP_ITF" title="${LTE_X_MMMM_MMELIST2_UP_ITF_title }" class="border border-box" disabled
						onblur="createMML();" js_regex="no_zh">
						<option value="eth2">WAN(DefaultValue)</option>
						<option value="ipsec">ipsec</option>
					</select>
					<div id="mme2_select_user_name_err" class="errSpan" style="display: block;margin-top: 5px;">
						
					</div>
				</li>
			</c:if>
		</div>
	</div>
</ul>

<script type="text/javascript"> 
	function addMmeIP(type) {
		var types = {
				1: '#mme1_ip',
				2: '#mme2_ip'
			},
			cSel = {
				1: '#mme1_select',
				2: '#mme2_select'
			},
			ctlctn = $(cSel[type]+'_control'),
			userctn = $(cSel[type]+'_user'),
			ipctn = $(types[type]+'_ctn'),
			input = $(types[type]+'_input'),
			hideInput = $(types[type]+'_hidden'),
			ipStr = input.val().trim(),
			span = ['<span class="ip-item-span">',
						'<span class="ip-item-text">',
						ipStr,
						'</span>',
						'<i class="el-icon el-icon-operation-delete" onclick="delIPSpan(this,'+type+')"></i>',
					'</span>'],
			bool = false;
		
		bool = validIPCanAdd(ipStr,ipctn);
		if(bool) {
			ipctn.append($(span.join('')));
			ctlctn.attr('disabled',false);
			userctn.attr('disabled',false);
			updateIPHidden(ipctn,hideInput);
		}else {
			ipctn.parent().addClass('invalid-ip-tips');
			setTimeout(function(){
				ipctn.parent().removeClass('invalid-ip-tips');
			},5000);
		}
	}
	function updateIPHidden(ipctn,hideInput) {
		var ips = Array.from(ipctn.find('.ip-item-text')).map(function(item){
				return $(item).text();
			});
		hideInput.val(ips.join(','));
		createMML();
	}
	function updateControlPlane() {
		var s1 = $('#mme1_select_control').val(),
			s2 = $('#mme2_select_control').val(),
			txt1 = $('#mme1_select_control option:selected').text(),
			txt2 = $('#mme2_select_control option:selected').text(),
			ctns = $('#mme1_select_control, #mme2_select_control'),
			tips = $('#mme1_control_tips, #mme2_control_tips');
		
		if(txt1 != txt2 ||  txt1=='--Unbound--' || txt2=='--Unbound--') {
			tips.hide();
			ctns.removeClass('err_border');
		}else {
			tips.show();
			ctns.addClass('err_border');
		}
		
		if(s1||s2) {
			var valTxt = s2+s1;
			
			if(s1 && s2) {
				valTxt = s2+','+s1;
			}
			
			$('#mme_control_hidden').val(valTxt);
		}else $('#mme_control_hidden').val('');
		
		createMML();
	}
	function delIPSpan(span,type) {
		var types = {
				1: '#mme1_ip',
				2: '#mme2_ip'
			},
			cSel = {
				1: '#mme1_select',
				2: '#mme2_select'
			},
			ctlctn = $(cSel[type]+'_control'),
			userctn = $(cSel[type]+'_user'),
			ipctn = $(types[type]+'_ctn'),
			hideInput = $(types[type]+'_hidden');
		
		$(span).parent().remove();
		updateIPHidden(ipctn,hideInput);
		// 无ip时，禁用接口绑定
		if(ipctn.find('.ip-item-span').length==0) {
			ctlctn.val('').attr('disabled',true);
			userctn.attr('disabled',true);
			updateControlPlane();
		}
	}
	function validIPCanAdd(ipStr,ctn) {
		return isIPv4(ipStr) && checkIpLength(ctn)<16 && !checkIpExist(ipStr);
	}
	function checkIpLength(ctn) {
		var items = ctn.find('.ip-item-text');
		return items.length;
	}
	function checkIpExist(ipStr) {
		var list = [],
			mme1IPs = Array.from($('#mme1_ip_ctn .ip-item-text')),
			mme2IPs = Array.from($('#mme2_ip_ctn .ip-item-text'));
		(mme1IPs.concat(mme2IPs)).map(function(item){
			list.push($(item).text().trim());
		})
		
		return list.includes(ipStr.trim());
	}
</script>  


