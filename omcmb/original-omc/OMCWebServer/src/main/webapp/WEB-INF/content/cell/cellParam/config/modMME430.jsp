<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>

<style>
	.reSetConfirm + .messager-button{
		padding-top:20px;
	}

	.mmeip-plmn-item {
		display: inline-block;
		padding: 2px 5px;
		margin: 0 5px 5px 0;
		border: 1px solid #4d84ff;
	}

	.mmeip-plmn-item .mmeip {
		display: inline-block;
		min-width: 95px;
		padding-bottom: 2px 0px;
		border-right: 1px dotted #4d84ff;
	}
	.mmeip-plmn-item .plmn {
		display: inline-block;
		min-width: 40px;
		padding: 2px;
	}
</style>

<ul id="paramNodesUl" class="paramNodesUl" style="height: 100%;">
	<div id="oneMMEPool">
		<li>
			<label for="LTE_POOL_MME_LIST1_name">${LTE_SIGLINK_SERVER_LIST_name }</label>
			<div style="display: flex;">
				<input id="LTE_POOL_MME_LIST1_name" name="LTE_POOL_MME_LIST1" title="${LTE_POOL_MME_LIST1_title }" class="border border-box" 
					min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);" js_regex="no_zh"/>
				<div style="padding: 5px;border: 1px solid #dedfe6;border-right: none;border-left: none;">PLMN</div>
				<select id="LTE_POOL_MME_LIST1_select" style="min-width: 80px;">
					<option>12345</option>
				</select>

				<input id="LTE_POOL_MME_LIST1_hide" name="" type="hidden"/>
				<i class="el-icon el-icon-plus" style="margin: 5px 0 0 15px;" onclick="bindPlmn()"></i>
			</div>
			<div id="LTE_POOL_MME_LIST1_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
				${LTE_POOL_MME_LIST1_title }
			</div>
				
			<div id="mme_plmn_list" style="margin-top: 5px;margin-left: 200px;min-width: 420px;">

			</div>
	    </li>
	</div>
</ul>

<script type="text/javascript"> 
    function bindPlmn() {
		var ipVal = $('#LTE_POOL_MME_LIST1_name').val(),
			plmnVal = $('#LTE_POOL_MME_LIST1_select').val();

		if(isIPv4(ipVal) && plmnVal && !isMMEPLMNExist(ipVal, plmnVal) && isLessThan16()) {
			var hideVal = $('#LTE_POOL_MME_LIST1_hide').val();

			if(hideVal) {
				hideVal += ';' + ipVal + ',' + plmnVal;
			}else {
				hideVal = ipVal + ',' + plmnVal;
			}

			appendBind(ipVal, plmnVal);
			$('#LTE_POOL_MME_LIST1_hide').val(hideVal);
			$('#LTE_POOL_MME_LIST1_name_err').removeClass('redColor');
		}else {
			$('#LTE_POOL_MME_LIST1_name_err').addClass('redColor');
		}
	}

	function appendBind(mmeIp, plmn) {
		var div = $('#mme_plmn_list'),
			itemStr = [
				'<div class="mmeip-plmn-item">',
					'<span class="mmeip">' + mmeIp + '</span>',
					'<span class="plmn">PLMN: ' + plmn + '</span>',
					'<span class="el-icon el-icon-close" style="font-size: 12px;" onclick="removeBind(this, &quot;'+mmeIp+'&quot;, '+plmn+')"></span>',
				'</div>'
			].join(' '),
			$item = $(itemStr);

		div.append($item);
		createMML();
	}

	function removeBind(item, mmeIp, plmn) {
		var hideVal = $('#LTE_POOL_MME_LIST1_hide').val(),
			list = hideVal.split(';');

		list.remove(mmeIp+','+plmn);
		$('#LTE_POOL_MME_LIST1_hide').val(list.join(';'));
		$(item).parent().remove();
		createMML();
	}

	function isMMEPLMNExist(mmeIp, plmn) {
		var bool = false,
			hideVal = $('#LTE_POOL_MME_LIST1_hide').val(),
			list = hideVal.split(';');

		if(list.includes(mmeIp + ',' + plmn)) {
			bool = true;
		}

		return bool;
	}

	function isLessThan16() {
		var hideVal = $('#LTE_POOL_MME_LIST1_hide').val()||'',
			list = hideVal.split(';');
		
		return list.length < 16;
	}
</script>  





