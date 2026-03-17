<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>

<style type="text/css">
	.switch { background-color: #66CC66; margin-left: 0px; float: none; }
	.optionDetailsStyle { position: relative; }
	.optionDetailsStyle.readonly::before { content: ''; display: inline-block; width: 100%; height: 100%; position: absolute; top: 0; left: 0; z-index: 1000; }
	.optionDetailsStyle.readonly input, .optionDetailsStyle.readonly .textbox { background-color: #F1F1F1; }
	.optionDetailsStyle.readonly input[type=checkbox] { opacity: 0.5; }
</style>

<!-- 定时报表 -->
<div id="kpiQueryDiv" style='height:100%'>
	<div class="slidebarTitleDiv">
		<div id="addTemplateTitle" style="display:inline-block;"><%=rb.getString("DingShiBaoBiao")%></div>
		<a class="el-icon el-icon-close slideIcon" onclick="closeReportTemplateDiv()" style="position: absolute; top:5px; right: 20px;font-size: 30px;"></a>
	</div>
	<div id="reportTemplateDiv_body" class="slideBody" style="height:93%">
		<form id="timing_report_form" style="background:#FFF;padding:40px;">
			<div style="margin-bottom:20px;">		
	 			<input id="report_kpi_temp_id" type="hidden" name="tempId"/>
	            <input id="report_kpi_status" type="hidden" name="reportStatus"/>
	            <div>
	            	<span class="inputTittleCss" style="display:inline;margin-right:8px;">Enable</span>
	               	<div class='switch' onclick="switchReportStatus(this)" style='vertical-align:middle;'>
						<div id='self_config_switch' isopen='true'  class='btnn' style='left:24px;'></div>
					</div>
			    </div>
			</div>
			<div id="period_setting_div">
				<label class="inputTittleCss"><%=rb.getString("ZhouQi")%></label>
				<input id="report_temp_time_granularity" type="hidden" name="reportPeriod">
				<div style="border: 1px solid #ddd;padding:10px 20px;display:inline-block;" id="report_period_setting">
					<span>
						<input type="checkbox" id="day_report_status" style='vertical-align: middle;' value="day"/><label for="day_report_status"> <%=rb.getString("Tian")%></label>
					</span>
					<span style="margin-left: 60px;">
						<input type="checkbox" id="houre_report_status" style='vertical-align: middle;' value="houre"/><label for="houre_report_status"> <%=rb.getString("XiaoShi")%></label>
					</span>
					<span style="margin-left: 60px;" v-show='curReportEnbGnbEgwType == "0"'>
						<input type="checkbox" id="minute_report_status" style='vertical-align: middle;' value="15Min"/><label for="minute_report_status"> 15Min</label>
					</span>
				</div>
				<div class="errorText"> <%=rb.getString("QingXuanZeZhouQi")%></div>
			</div>
			<div style="margin-bottom:15px;">
	            <div style="display: flex;">
	                <input type="checkbox" id='mail_report_status' v-model='emailShow' name="mailStatus" value="1" @change="change"/> 
					<label class="inputTittleCss" for="mail_report_status"> <%=rb.getString("YouXiangKaiGuan")%></label>
				</div>
			</div>
			<div style="margin-bottom:15px;">
				<label class="inputTittleCss"><%=rb.getString("FaSongShiJian")%></label>
				<select id="temp_time_collect" name="reportTime" class="easyui-combobox inputDivCss" data-options="editable:false" style="height: 26px;width:200px;">
	                <option value="0">00:00</option>
	                <option value="1">01:00</option>
	                <option value="2">02:00</option>
	                <option value="3">03:00</option>
	                <option value="4">04:00</option>
	                <option value="5">05:00</option>
	                <option value="6">06:00</option>
	                <option value="7">07:00</option>
	                <option value="8">08:00</option>
	                <option value="9">09:00</option>
	                <option value="10">10:00</option>
	                <option value="11">11:00</option>
	                <option value="12">12:00</option>
	                <option value="13">13:00</option>
	                <option value="14">14:00</option>
	                <option value="15">15:00</option>
	                <option value="16">16:00</option>
	                <option value="17">17:00</option>
	                <option value="18">18:00</option>
	                <option value="19">19:00</option>
	                <option value="20">20:00</option>
	                <option value="21">21:00</option>
	                <option value="22">22:00</option>
	                <option value="23">23:00</option>
	            </select>
			</div>			
			<div>
				<label class="inputTittleCss"><%=rb.getString("YouXiang")%></label>
				<input id="report_temp_mailsInput" name="mailAddress" type="hidden" value="">
				<textarea id="temp_mailsInput" onblur="isEmail(this);" class="border" style="width:450px;height:80px;" 
					oldValue="" placeholder="<%=rb.getString("YouXiangShuRuYaoQiu")%>"></textarea>
				<div class="tipsText"><%=rb.getString("YouXiangDiZhiTiShi")%></div>
				<div id="mailError" style="color:red;"></div>
			 </div>
		</form>
	</div>	
</div>
<div class="slideFooter">
	<span class="el-button el-button--primary" onclick="reportTemplateSubmit()"><%=rb.getString("QueDing")%></span>
    <span class="el-button" onclick="closeReportTemplateDiv()"><%=rb.getString("QuXiao")%></span>
</div>
	
<script type="text/javascript">
	var curReportEnbGnbEgwType = sessionStorage.getItem('reportTemplateEnbGnbOrEgw');
	
	var settingsVue = new Vue({
		el:'#kpiQueryDiv',
		data:{
			emailShow:null
		},
		methods:{
			change(value){
				var vm = this;
				if(vm.emailShow){
					$('#temp_mailsInput').attr('disabled',false);
					//判断当前是否填写 setting 中的email 判断是否可以新建
					axios.post("${ctx}/cell/fault/queryHasSettingEmailConfig.action").then(function(response){
						if(response.data.result == '2'){
							vm.$alert('<%=rb.getString("SheZhiYouXiangTipsKPI")%>',"<%=rb.getString("TiShi")%>",{
								confirmButtonText:'<%=rb.getString("QueDing")%>',
							})							
							return false;
						}
						
						if(response.data.result == '3'){
							vm.$alert('<%=rb.getString("MeiYouSheZhiYouXiangKPI")%>',"<%=rb.getString("TiShi")%>",{
								confirmButtonText:'<%=rb.getString("QueDing")%>',
							})
							return false;
						}						
					})					
				}else{
					$('#temp_mailsInput').val('');
					$('#temp_mailsInput').attr('disabled',true);
					$('#mailError').html('');
				}
			}
		}
	})
	
	$(function(){
	    closeLoading();
	
	    // 周期设置事件绑定
	    $('#report_period_setting input').on('click',function(){
	    	var list = $('#report_period_setting input:checked'),
	    		values = Array.from(list).map(function(item){ return item.value;}),
	    		status = $('#report_kpi_status').val();
	    	$('#report_temp_time_granularity').val(values.join(','));
	    	
	    	if(values.join(',') || status != '1') {
	    		$('#period_setting_div .errorText').css('visibility','hidden');
	    	}else {
	    		$('#period_setting_div .errorText').css('visibility','visible');
	    	}
	    })
	    
	    $('#report_kpi_temp_id').val(globleTempId);
	    
	    initReportForm();
	});
	
	//初始化定时报表数据
	function initReportForm(){
		var curReportInfoUrl = '';
		
		if(curReportEnbGnbEgwType == '0'){
			curReportInfoUrl = '${ctx}/pm/template/getRegularReportInfo.action';
		}else if(curReportEnbGnbEgwType == '1'){
			curReportInfoUrl = '${ctx}/gnb/pm/template/getRegularReportInfo.action';
		}else{
			curReportInfoUrl = '${ctx}/egw/pm/template/getRegularReportInfo.action';
		}
		
		$.post(curReportInfoUrl, {tempId: globleTempId, timeZone: timeZone}, function(data) {
			if(data) {
				$('#report_kpi_status').val(data['reportStatus']);
				if(data['reportStatus']!= '1') {// 初始化开关状态
					self_config_switch.click();
				}
				
				if(data['reportPeriod']) {
					$('#report_temp_time_granularity').val(data['reportPeriod']);
					data['reportPeriod'].split(',').map(function(item){
						$('#period_setting_div input[value="'+item+'"]').prop('checked',true);
					})
				}
				
				if(data['reportTime']) {
					$('#temp_time_collect').combobox('setValue', data['reportTime']);
				}
				
				if(data['mailStatus'] == '1') {
					mail_report_status.click();
					$('#temp_mailsInput').attr('disabled',false);
				}else{
					$('#temp_mailsInput').val('');
					$('#temp_mailsInput').attr('disabled',true);
					$('#mailError').html('');
				}
				 
				if(data['mailAddress']) {
					$('#temp_mailsInput').val(data['mailAddress']);
					$('#report_temp_mailsInput').val(data['mailAddress']);
				}
			}
		}, "json"); 
	}
	
	//定时报表开关控制
	function switchReportStatus(ele){
		var $dom = $('#report_kpi_status');
		if ($(ele).children().attr('isopen') == 'false') {
			$(ele).children().attr('isopen','true').animate({left:'24px'},100);
			$(ele).css('background-color','#66CC66');
			$dom.val(1);			
		} else {			
			$(ele).children().attr('isopen','false').animate({left:'1px'},100);
	        $(ele).css('background-color','#838383');
	        $dom.val(0);
	       	$('#temp_mailsInput').val('');
			$('#mailError').html('');
		}
	}
	
	function isEmail(ele){
	    var addresses = $(ele).val();
	    var checked  = $('#mail_report_status').prop('checked');
	    var config_switch = $('#self_config_switch').attr('isopen');
		addresses = addresses.replace(/\s/g,"").trim();
		if(checked && config_switch == 'true'){
			if ( addresses ){			
				if(addresses.lastIndexOf(";")+1 == addresses.length){
					addresses = addresses.substring(0,addresses.length-1); 
				}
				var add = addresses.split(";");
				for (var i = 0;i < add.length; i++){
					var str = add[i];
					if (validateEmail(str)){
						$(ele).next().next().html("");
						
					}else {
						$(ele).next().next().html("<%=rb.getString("YouXiangGeShiCuoWu")%>");
						$(ele).next().next().css("color","red");
						return;
					} 
				}
			}else {
				if ( checked ){
					$(ele).next().next().html("<%=rb.getString("QingShuRuYouXiang")%>");
					$(ele).next().next().css("color","red");
					return;
				} else {
					$(ele).next().next().html("");
				}				
			} 
		}		
	}	
	
	// 定时报表保存
	function reportTemplateSubmit(){
		var params = $('#timing_report_form').serializeJSON(), curSaveReportInfoUrl = '';
		formateFormData(params);
		if( !validReportForm(params) ) return;

		if(curReportEnbGnbEgwType == '0'){
			curSaveReportInfoUrl = '${ctx}/pm/template/updateRegularReport.action';
		}else if(curReportEnbGnbEgwType == '1'){
			curSaveReportInfoUrl = '${ctx}/gnb/pm/template/updateRegularReport.action';
		}else{
			curSaveReportInfoUrl = '${ctx}/egw/pm/template/updateRegularReport.action';
		}
		
		$.post(curSaveReportInfoUrl, params, function(data) {
			if(data){
				if (data["success"]) {
					$("#kpiTemplateDatagrid").datagrid("reload");
					showMsg('success_msg','<%=rb.getString("BaoCunChengGong")%>');
					setTimeout(closeReportTemplateDiv,1000);
				} else {
					showMsg('error_msg',data.message);
				}
			}else {
				showMsg('error_msg','<%=rb.getString("CuoWu")%>');
			}
		}, "json"); 
	}
	
	/** 
	* 表单数据提交前修正
	* @param params[string] 参数
	**/
	function formateFormData(params){
		var mailAddress = $('#temp_mailsInput').val();
		if(!params.mailStatus) params.mailStatus = '0';
		if(mailAddress) {
			params.mailAddress = mailAddress;
			$('#report_temp_mailsInput').val(mailAddress);
			$('#temp_mailsInput').blur();
		}else{
			params.mailAddress = '';
		}
		params.timeZone = timeZone;
	}
	
	/** 
	* 表单校验
	* @param params[string] 参数
	**/
	function validReportForm(params) {
		var bool = true;
		
		if ( params.reportStatus == 1 && params.reportPeriod == ""){
			$('#period_setting_div .errorText').css('visibility','visible');
			bool = false;
		}else {
			$('#period_setting_div .errorText').css('visibility','hidden');
		}
		
		if ( params.reportStatus == 1 && params.mailStatus == 1 && params.mailAddress == ""){
			$("#mailError").text('<%=rb.getString("QingShuRuYouXiang")%>');
			bool = false;
		}else if($("#mailError").text().length != 0){
			bool = false;
		}else{
			$("#mailError").text('')
		}
		
		return bool;
	}
</script>