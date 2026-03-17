<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
.borderPage {
	border:1px solid #d5dcec;
	border-radius:10px;
	height:100%;
	background:#fff;
}
.limitForm {
	padding:30px;
}
.limitForm .el-form-item__label {
	line-height:initial;
}
.error-color {
	color:#f56c6c;
}
#enbLimitPage .el-form-item__error {
	top:unset;
	left:0;
}
</style>

<div id="enbLimitPage" class="borderPage" style='display:flex;flex-direction:column;'>
	<div class='el-card__body'>
		<el-form ref="form" :model="limitForm" :rules="limitFormRule" class="limitForm">
			<el-form-item label="<%=rb.getString("LiuLiangKaiGuan")%>" prop="limitSwitch">
				<el-switch v-model="limitForm.limitSwitch" active-value="1" inactive-value="0"></el-switch>
			</el-form-item>
			<el-form-item label="<%=rb.getString("LiuLiangXianE")%>" prop="amount" label-width="130px">
				<el-input v-model="limitForm.amount" @blur="limitBlur">
					<template slot="append">MByte</template>
				</el-input>
				<el-input v-if="false"></el-input>
			</el-form-item>
			<el-form-item label="<%=rb.getString("YiYongLiuLiang")%>" >
				<span id="used_flow" style="display: inline-block;width: 140px;text-align: right;font-weight: bold;color:#EF6262;font-size: 20px;"></span>
				<span style="padding: 5px;">MByte</span>
				<button :disabled="disabledReset" class="el-button el-button--mini"><span onclick="saveLimitation(true)"><i class="el-icon el-icon-operation-reset"></i> <%=rb.getString("ChaXunChongZhi")%></span></button>
				<div v-if="showTip" class="result-tips error-color" style="padding-left: 60px;">{{overTip}}</div>
			</el-form-item>

			<div v-if="stateCode == 'send' || stateCode == 'unsend'" style="margin-bottom: 0px;">
				<span class="result-tips warning-color">
					<i class="el-icon el-icon-circle-warning" style="font-size: 20px;">{{tipText}}</i> 
				</span>
			</div>
		</el-form>
	</div>
	
	<div class='el-card__footer'>
		<el-button type="primary" @click="saveLimitation"><%=rb.getString("QueDing")%></el-button>
		<el-button @click="closeEnbLimitPage"><%=rb.getString("QuXiao")%></el-button>
	</div>
</div>

<script>
var enbLimitPage = new Vue({
	el: '#enbLimitPage', 
	data() {
		var vm = this;
		var validAmount = function(rule ,value,cb){
			if(vm.limitForm.limitSwitch == '0'){
				cb();
			}else if(!value){
				cb('<%=rb.getString("QingShuRu")%><%=rb.getString("LiuLiangXianE")%>')
			}else {
				cb()
			}
		};
		return {
            enbSelectedRow:{},
			code:'',
			limitForm:{
				"limitSwitch":0,
				"amount":"",
				"used_flow":""
			},
			limitFormRule:{
				amount:[{validator:validAmount,trigger:'blur'}]
			},
			tipText:'',
			overTip:'<%=rb.getString("LiuLiangYiDaXianEr")%>',
			showTip:false,
			stateCode:'',
			status:'',
			disabledReset:true
		};
	},
	methods: {
		init(row,code,sn,status){
			var vm = this;
            vm.enbSelectedRow = row;
			vm.code = code;
			vm.status = status;
			var params = {},
			bool = false,
			params = {'small_cell_code': vm.code};
			
			axios.post('${ctx}/cell/kuailte/getCellLimitationInfo.action',stringify(params)).then(function(response){
				let data = response.data;
				
				vm.limitForm.limitSwitch = data.traffic_limit_enable  || '0';
				vm.limitForm.amount = data.traffic_limit_amount;
				vm.limitForm.used_flow = data.traffic_usage||'';
				
				var amount = data.traffic_limit_amount,
				usage = data.traffic_usage||'';
				
				var statusList = ['send','unsend'],status = data.status;
				vm.stateCode = statusList[status];
				
				if(vm.stateCode){
					vm.tipText = stateCode =='send' ? '<%=rb.getString("MingLingYiFaSong")%>' :'<%=rb.getString("MingLingFaSong")%>'
				}
				
				if(amount && amount-usage<0) {
					vm.showTip = true;
				}
				vm.limitBlur(); 
	
				
			}).catch(function(error){})
					
		},
		limitBlur(){
			var vm = this;
			var amount = vm.limitForm.amount,
				usage = vm.limitForm.used_flow,
				online = vm.status == 'On';
			
			if(usage && online) {
				vm.disabledReset = false
			}
			
			if(amount && amount-usage<0) {
				vm.showTip = true;
			}
		},
		saveLimitation(isReset){
			var vm = this;
			var params = {
					small_cell_code: vm.code,
					traffic_limit_enable: vm.limitForm.limitSwitch,
					traffic_limit_amount: vm.limitForm.amount,
					traffic_limit_reset: isReset?1:0,
					traffic_usage: vm.limitForm.used_flow
				};
			
			vm.$refs.form.validate((valid) => {
	    		if(valid){
											
					var url = "${ctx}/cell/kuailte/setCellLimitationInfo.action";
					axios.post(url,stringify(params)).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message({
								message:message,
								type:'success',
							})
						}else{
							vm.$message.error(data["message"])
						}
						
					}).catch(function(error){})
				}
			})
		},
        closeEnbLimitPage(){
			var vm = this;
			settingVue.closeSetting();
		}
    },
	mounted() {
		eventBus.$off("enb-data").$on("enb-data",this.init)
	}
});

</script>
