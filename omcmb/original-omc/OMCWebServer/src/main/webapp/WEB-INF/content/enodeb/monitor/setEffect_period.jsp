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
.periodForm {
	padding:30px;
}
.periodForm .el-form-item__label {
	line-height:initial;
}
.tipText {
	font-size:14px;
	color:#999;
}
#enbPeriodPage .el-form-item__error{
	width: unset;
	top: unset;
	left: unset;
}
</style>

<div id="enbPeriodPage" class="borderPage" style='display:flex;flex-direction:column;'>
	<div class='el-card__body'>
		<el-form ref="form" :model="periodForm" :rules="periodFormRule" label-position="left" label-width="125" class="periodForm">
			<el-form-item label="<%=rb.getString("GongNengKaiGuan")%>" prop="validitySwitch">
				<el-switch v-model="periodForm.validitySwitch" @change="changeSwitch" active-value="1" inactive-value="0"></el-switch>
			</el-form-item>
			<el-form-item label="<%=rb.getString("YouXiaoQi")%>" prop="validity" style='margin-bottom: 22px;'>
				<el-date-picker 
					v-model="periodForm.validity" 
					:disabled="dateDisable" 
					value-format="yyyy-MM-dd"
					 type="date" 
					:picker-options="pickerOptions" 
					@change="validityHour">
				</el-date-picker>
				
			</el-form-item>
			<el-form-item>
				<span v-if="expire"><%=rb.getString("YouXiaoQiYiGuoQingChongXinSheZhi")%></span>
				<p class="tipText"><%=rb.getString("YouXiaoShiChang")%>： {{validity_hour}}  <%=rb.getString("XiaoShi")%></p>
			</el-form-item>
		</el-form>
	</div>
	
	<div class='el-card__footer'>
		<el-button type="primary" @click="saveValidityPeriod"><%=rb.getString("QueDing")%></el-button>
		<el-button @click="init"><%=rb.getString("QuXiao")%></el-button>
	</div>
</div>
<script>
var enbPeriodPage = new Vue({
	el: '#enbPeriodPage', 
	data() {
		var vm = this;
		var validDate= function(rule ,value,cb){
			
			if(vm.periodForm.validitySwitch == '0'){
				cb();
			}else if(value == '' || value == null){
				cb('<%=rb.getString("QingXuanZeDaoQiShiJian")%>')
			}else {
				cb()
			}
		};
		return {
            enbSelectedRow:{},
			timeVal:[new Date()- 3600*1000*24*3, new Date],
			pickerOptions:{
				disabledDate(time){
					return time.getTime()<Date.now() - 1*24*3600*1000;
				}
			},
			periodForm:{
				"validitySwitch":0,
				//"validity":[new Date()- 3600*1000*24*3, new Date]
				"validity":''
			},
			dateDisable:true,
			periodFormRule:{
				validity:[{validator:validDate,trigger:'blur'}]
			},
			enbCode:'',
			expire:false,
			validity_hour:''
		};
	},
	methods: {
		init(row,code,sn,status){
			var vm = this;
			vm.enbCode = code;
            vm.enbSelectedRow = row;
			var params = {
					smallCellCode : vm.enbCode,
					timeZone : timeZone
			}

			axios.post("${ctx}/cell/cpeinfos/getCellValidityInfo.action",stringify(params)).then(function(response){
				let data = response.data;
				
				vm.periodForm.validitySwitch = data.validitySwitch ? data.validitySwitch : '0';
				vm.periodForm.validity = data.validity == null ? '' : data.validity.substring(0,10);
				vm.validity_hour = data.validityHour;
				vm.expire = data.expire == '1'? true :false
				vm.dateDisable = data.validitySwitch == '1' ? false : true;
				
			}).catch(function(error){})
			
		},
		changeSwitch(val){
			var vm = this;
			vm.dateDisable = val == '1' ? false : true;
		},
		validityHour(val){
			var d = new Date(val);
			var dateTime = d.getFullYear() + '-' + (d.getMonth() +1) + '-' + d.getDate() + ' 23:59:59';
			var validity_time = new Date(dateTime);
			var current_time = new Date(gloableTime);
			var hour = (validity_time.getTime()-current_time.getTime())/(1000*3600);
			this.validity_hour = parseInt(hour);
		},
		saveValidityPeriod(){
			var vm = this;
			
			vm.$refs.form.validate((valid) => {
	    		if(valid){
					var params = {
							smallCellCode: vm.enbCode,
							validity: vm.periodForm.validity,
							timeZone: timeZone,
							validitySwitch: vm.periodForm.validitySwitch
						}
											
					var url = "${ctx}/cell/cpeinfos/updateCellValidity.action";
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
		    
	},
	mounted() {
		eventBus.$off("enb-data").$on("enb-data",this.init)
	}
});

</script>