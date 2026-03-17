<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#gsmSettingBasicPage{
	height: 100%;
	width: 100%;
}
#gsmSettingBasicPage .itemMainBoxCls{
	border-radius:10px;
	background:#fff;
	height:100%;
	width: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
}
#gsmSettingBasicPage .itemMainBoxTitle {
	height:36px;
	padding-left: 20px;
    line-height: 36px;
	font-size:14px;
	font-weight:bold;
	border-bottom: 1px solid #E9E9E9;
}
#gsmSettingBasicPage .itemMainBoxCenter{
	width: 100%;
	flex:1;
	overflow: auto;
}
#gsmSettingBasicPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
    border-top : 1px solid #E9E9E9;
	height:48px;
	background-color: #FFFFFF;
    box-sizing: border-box;
    width: 100%;
	padding-left: 20px;
}
#gsmSettingBasicPage .el-form-item{
	margin-bottom: 20px;
}
#gsmSettingBasicPage .errorBoxCls{
	color:red;
	font-size:10px;
}
#gsmSettingBasicPage .el-form-item .el-form-item__label{
	font-size: 12px;
}
#gsmSettingBasicPage .itemMainBoxCls .itemMainBoxCenter .el-collapse-item__wrap{
    border-bottom:none;
    padding-left:40px;
}
</style>

<div id="gsmSettingBasicPage">
	<div class="itemMainBoxCls">
		<div class="itemMainBoxTitle">
			<%=rb.getString("ENBJiChuPeiZhi")%>
		</div>
		<div class="itemMainBoxCenter">
			<el-form :model='ruleForm' ref="ruleForm" :rules="rules" label-position="top">
				<el-collapse v-model="activeCollapse">
					<el-collapse-item name="quickSetting">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold"><%=rb.getString("KuaiSuSheZhi")%></span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div style="display:flex;margin-left:16px;flex-wrap: wrap">
								<el-form-item prop='bscName' style="width:40%;min-width:400px;" label="<%=rb.getString("BSCMingCheng")%>" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.bscName' maxlength="48" style="width:150px;padding-top:5px;">
										<template slot="append">Length：0~48</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='' style="width:40%;min-width:400px;display:none" label-width="160px">
									<el-input></el-input>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
				</el-collapse>
			</el-form>
		</div>
		<div class='itemMainBoxFooter'>
			<el-button type="primary" @click="settingsSubmit"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="closeSettings" ><%=rb.getString("QuXiao")%></el-button>
		</div>
	</div>
</div>

<script>
var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
	regKey = /^[A-Fa-f0-9]{32}$/,
	regNumber = /^[0-9]{15}$/;
var gsmSettingBasicPageVue = new Vue({
	el: '#gsmSettingBasicPage', 
	data() {
		var vm = this,
			validateRange = (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var mag = rule.mag;
				var isRequired = rule.isRequired;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;

				if(value == '' || value == undefined || value == null){
					if(isRequired){
						callback(new Error(mag))
					}else{
						callback();
					}
				}else{
					if(reg.test(value) && value >= min && value <= max){
						callback();
					}else{
						callback(new Error(mag))
					}
				}
			};
		return {
			activeCollapse:['quickSetting'],
			rowDataInfo: [],
			smallCellCode:'',
			ruleForm:{
                bscName:'',
			},
			rules:{},
		};
	},
	computed: {},
	watch: {},
	methods: {
		init(row){
			var vm = this;
			vm.rowDataInfo = row;
			vm.smallCellCode = row.small_cell_code;
			vm.ruleForm.bscName = row.host_name;
			initForm(vm.$refs.ruleForm);
		},
		// 判断是否为空
		isNull(val){
			if(val==undefined || val == null || val =="") return true;
			else return false;
		},
		// 验证输入的是否是整数
		isInteger(str) {
			if(str.length==0){
				return false;
			}
			var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
			if(!reg.test(str)){
				return false;
			}
			return true;  
		},
		settingsSubmit(){
			var vm = this,
				params = {
					gsmCode: vm.smallCellCode,
					bscName: vm.ruleForm.bscName,
				},
				url = '${ctx}/gsm/config/setBasicConfig.action',
				isChanged = isFormChanged(vm.$refs.ruleForm);

			if(!isChanged){
				showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
				return;
			}
			vm.$refs.ruleForm.validate(function(valid){
				if(valid) {
					$('#gnbSetting_main').addClass('loading');
					var saveParams = JSON.stringify(params);
					axios.post(url,saveParams,{headers:{'Content-Type':'application/json;charset=utf-8'}}).then(res=>{
						var data = res.data;
						if(data["success"]){
							vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});
							vm.closeSettings();
						}else{
							vm.$message.error(data["message"])
						}
						$('#gnbSetting_main').removeClass('loading');
					})
				}
			});
		},
		closeSettings(){
            eventBus.$emit('gsm-close-setting');
		},
	},
	mounted() {
		eventBus.$off("gsm-data").$on("gsm-data",this.init)
	}
});

</script>
