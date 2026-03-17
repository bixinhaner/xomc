<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8" %>

<style>
	.limitation-form > div {
		margin-bottom: 30px;
	}
	.limitation-title {
		display: inline-block;
		width: 95px;
	}
</style>

<div id="limitationDG" title="<%=rb.getString("LiuLiangXianZhi")%>" class="easyui-dialog" style="width: 500px; height: 320px;" data-options="modal: true,closed: true">
	<div class="flex-ctn" style="height: 99%;">
		<div style="flex: auto;overflow: auto;padding: 15px 30px;" class="limitation-form">
			<div style="display: flex;">
				<span class="limitation-title"><%=rb.getString("LiuLiangKaiGuan")%></span>
				<div class='switch' onclick="OpenLimitationControl(this)" style="float: none;margin-left: 0px;">
					<div id='limit_switch' isopen='false' oldValue="" class='btnn' style='left:1px;'></div>
				</div>
			</div>
			<div>
				<span class="limitation-title"><%=rb.getString("LiuLiangXianE")%></span>
				<input id="limit_flow" class="easyui-numberbox" type="number" data-options="min:0,events: {blur: limitBlur}"/>
				<span style="padding: 5px;">MByte</span>
				<input id="traffic_usage" type="hidden" />
				<input id="small_cell_code" type="hidden" />
			</div>
			<div>
				<span class="limitation-title"><%=rb.getString("YiYongLiuLiang")%></span>
				<span id="used_flow" style="display: inline-block;width: 140px;text-align: right;font-weight: bold;color:#EF6262;font-size: 20px;"></span>
				<span style="padding: 5px;">MByte</span>
				<button id="limit_reset_bt" class="el-button el-button--mini"><span onclick="saveLimitation(true)"><i class="el-icon el-icon-operation-reset"></i> <%=rb.getString("ChaXunChongZhi")%></span></button>
				<div id="limit_tips_overflow" class="result-tips error-color" style="padding-left: 100px;">
					<%=rb.getString("LiuLiangYiDaXianEr")%>
				</div>
			</div>
			<div style="margin-bottom: 0px;">
				<span id="limit_tips_send" class="result-tips warning-color">
					<i class="el-icon el-icon-circle-warning" style="font-size: 20px;"></i> <%=rb.getString("MingLingYiFaSong")%>
				</span>
				<span id="limit_tips_unsend" class="result-tips warning-color">
					<i class="el-icon el-icon-circle-warning" style="font-size: 20px;"></i> <%=rb.getString("MingLingFaSong")%>
				</span>
			</div>
		</div>
		<div style="padding: 15px 30px;">
			<button class="el-button el-button--primary el-button--mini" onclick="saveLimitation()"><%=rb.getString("QueDing")%></button>
			<button class="el-button el-button--mini" onclick="$('#limitationDG').dialog('close');"><%=rb.getString("QuXiao")%></button>
		</div>
	</div>
</div>

<script>
	/**
	* 设置流量限制 -- 初始化
	* @param code{string}: 基站编码
	**/
	function setLimitation(code) {
		var params = {},
			bool = false,
			params = {'small_cell_code': code};

		$.ajax({
			url: '${ctx}/cell/kuailte/getCellLimitationInfo.action',
			data: params,
			type: 'post',
			dataType: 'json',
			success: function(data){
				var enable = data.traffic_limit_enable == '1',
					amount = data.traffic_limit_amount,
					usage = data.traffic_usage||'',
					statusList = ['send','unsend']
					status = data.status,
					stateCode = statusList[status],
					limitEnable = $('#limit_switch').attr('isopen') == 'true';
				
				if(enable != limitEnable) {
					OpenLimitationControl($('#limit_switch').parent())
				}
				
				$('#limit_flow').numberbox('setValue',amount);
				$('#small_cell_code').val(code);
				$('#traffic_usage').val(usage);
				$('#used_flow').text(usage);
				
				// set tips
				$('#limit_tips_send,#limit_tips_unsend,#limit_tips_overflow').hide();
				stateCode == 'send' && $('#limit_tips_send').show();
				stateCode == 'unsend' && $('#limit_tips_unsend').show();
				
				if(amount && amount-usage<0) {
					$('#limit_tips_overflow').show();
				}
				limitBlur();

				$('#limitationDG').dialog('open');
			}
		});
	}
	/**
	* 保存流量限制数据
	* @param isReset{boolean}: 是否重置
	**/
	function saveLimitation(isReset) {
		var params = {
				small_cell_code: $('#small_cell_code').val(),
				traffic_limit_enable: $('#limit_switch').attr('isopen') == 'true'?1:0,
				traffic_limit_amount: $('#limit_flow').numberbox('getValue'),
				traffic_limit_reset: isReset?1:0,
				traffic_usage: $('#traffic_usage').val()
			};
		
		$.ajax({
			url: '${ctx}/cell/kuailte/setCellLimitationInfo.action',
			data: params,
			type: 'post',
			dataType: 'json',
			success: function(data){
				if(data.success) {
					showMsg('success_msg',data.message);
					$('#limitationDG').dialog('close');
				}else {
					showMsg('error_msg',data.message);
				}
			}
		});
	}
	/**
	* 流量限制数据 -- 失去焦点事件
	* @param ev{event}: 鼠标事件
	**/
	function limitBlur(ev) {
		var code = $('#small_cell_code').val(),
			amount = $('#limit_flow').numberbox('getValue'),
			usage = $('#traffic_usage').val(),
			row = enbvm.selectedRow,
			online = row.connection_status == 'On';
		
		if(usage && online) {
			$('#limit_reset_bt').removeClass('disabled readonly');
		}else {
			$('#limit_reset_bt').addClass('disabled readonly');
		}
		if(amount && amount-usage<0) {
			$('#limit_tips_overflow').show();
		}else {
			$('#limit_tips_overflow').hide();
		}
	
	}
	/**
	* 展示流量限制浮层
	* @param ele{dom}: 浮层容器dom节点
	**/
	function OpenLimitationControl(ele){
		if ($(ele).children().attr('isopen') == 'false') {
			$(ele).children().attr('isopen','true').animate({left:'24px'},100);
			$(ele).css('background-color','#66CC66');
		} else {
			$(ele).children().attr('isopen','false').animate({left:'1px'},100);
			$(ele).css('background-color','#838383');
		}
	}
</script>